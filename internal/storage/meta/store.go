package meta

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var ErrNotFound = errors.New("not found")

type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type APIKey struct {
	ID, ProjectID, Name, TokenHash, Role string
	UserID                               string
	ExpiresAt                            *time.Time
	Revoked                              bool
	LastUsedAt                           *time.Time
}

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type WorkspaceMember struct {
	WorkspaceID string
	UserID      string
	Role        string
	CreatedAt   time.Time
}

type OAuthApp struct {
	ID          string    `json:"id"`
	ClientID    string    `json:"client_id"`
	ProjectID   string    `json:"project_id"`
	PublicKeyID string    `json:"public_key_id"`
	PublicKey   string    `json:"public_key,omitempty"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Store struct {
	db             *sql.DB
	lastUsedMu     sync.Mutex
	lastUsedMemory map[string]time.Time
}

var ErrAlreadyUsed = errors.New("OAuth JWT has already been used")

func NewStore(db *sql.DB) *Store { return &Store{db: db, lastUsedMemory: make(map[string]time.Time)} }
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func (s *Store) Migrate(ctx context.Context) error { return migrate(ctx, s.db) }
func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS projects (id TEXT PRIMARY KEY, name TEXT NOT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL); CREATE TABLE IF NOT EXISTS users (id TEXT PRIMARY KEY, email TEXT NOT NULL UNIQUE, name TEXT NOT NULL, password_hash TEXT NOT NULL, created_at INTEGER NOT NULL); CREATE TABLE IF NOT EXISTS workspace_members (workspace_id TEXT NOT NULL REFERENCES projects(id), user_id TEXT NOT NULL REFERENCES users(id), role TEXT NOT NULL DEFAULT 'member', created_at INTEGER NOT NULL, PRIMARY KEY(workspace_id,user_id)); CREATE TABLE IF NOT EXISTS user_sessions (token_hash TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id), workspace_id TEXT NOT NULL REFERENCES projects(id), expires_at INTEGER NOT NULL, created_at INTEGER NOT NULL); CREATE INDEX IF NOT EXISTS idx_sessions_user ON user_sessions(user_id); CREATE TABLE IF NOT EXISTS api_keys (id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), name TEXT NOT NULL, token_hash TEXT NOT NULL UNIQUE, role TEXT NOT NULL DEFAULT 'project', expires_at INTEGER, revoked INTEGER NOT NULL DEFAULT 0, last_used_at INTEGER); CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(token_hash); CREATE TABLE IF NOT EXISTS oauth_apps (id TEXT PRIMARY KEY, client_id TEXT NOT NULL UNIQUE, project_id TEXT NOT NULL REFERENCES projects(id), public_key_id TEXT NOT NULL, public_key TEXT NOT NULL, enabled INTEGER NOT NULL DEFAULT 1, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL); CREATE INDEX IF NOT EXISTS idx_oauth_apps_client_id ON oauth_apps(client_id); CREATE TABLE IF NOT EXISTS oauth_access_tokens (token_hash TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), expires_at INTEGER NOT NULL, created_at INTEGER NOT NULL); CREATE INDEX IF NOT EXISTS idx_oauth_access_tokens_expiry ON oauth_access_tokens(expires_at); CREATE TABLE IF NOT EXISTS oauth_jti (client_id TEXT NOT NULL, jti TEXT NOT NULL, expires_at INTEGER NOT NULL, created_at INTEGER NOT NULL, PRIMARY KEY(client_id,jti)); CREATE INDEX IF NOT EXISTS idx_oauth_jti_expiry ON oauth_jti(expires_at);`); err != nil {
		return err
	}
	var hasRole int
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(api_keys)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == "role" {
			hasRole = 1
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if hasRole == 0 {
		_, err = db.ExecContext(ctx, `ALTER TABLE api_keys ADD COLUMN role TEXT NOT NULL DEFAULT 'project'`)
	}
	return err
}

func (s *Store) CreateOAuthApp(ctx context.Context, app OAuthApp) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO oauth_apps(id,client_id,project_id,public_key_id,public_key,enabled,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, app.ID, app.ClientID, app.ProjectID, app.PublicKeyID, app.PublicKey, app.Enabled, app.CreatedAt.UTC().UnixMicro(), app.UpdatedAt.UTC().UnixMicro())
	return err
}

func (s *Store) OAuthAppByClientID(ctx context.Context, clientID string) (OAuthApp, error) {
	var app OAuthApp
	var enabled int
	var created, updated int64
	err := s.db.QueryRowContext(ctx, `SELECT id,client_id,project_id,public_key_id,public_key,enabled,created_at,updated_at FROM oauth_apps WHERE client_id=?`, clientID).Scan(&app.ID, &app.ClientID, &app.ProjectID, &app.PublicKeyID, &app.PublicKey, &enabled, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return app, ErrNotFound
	}
	if err != nil {
		return app, err
	}
	app.Enabled = enabled != 0
	app.CreatedAt = time.UnixMicro(created).UTC()
	app.UpdatedAt = time.UnixMicro(updated).UTC()
	return app, nil
}

func (s *Store) ListOAuthApps(ctx context.Context) ([]OAuthApp, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,client_id,project_id,public_key_id,public_key,enabled,created_at,updated_at FROM oauth_apps ORDER BY created_at,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []OAuthApp
	for rows.Next() {
		var app OAuthApp
		var enabled int
		var created, updated int64
		if err := rows.Scan(&app.ID, &app.ClientID, &app.ProjectID, &app.PublicKeyID, &app.PublicKey, &enabled, &created, &updated); err != nil {
			return nil, err
		}
		app.Enabled = enabled != 0
		app.CreatedAt = time.UnixMicro(created).UTC()
		app.UpdatedAt = time.UnixMicro(updated).UTC()
		result = append(result, app)
	}
	return result, rows.Err()
}

func (s *Store) CreateOAuthAccessToken(ctx context.Context, tokenHash, projectID string, expiresAt, createdAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO oauth_access_tokens(token_hash,project_id,expires_at,created_at) VALUES(?,?,?,?)`, tokenHash, projectID, expiresAt.UTC().UnixMicro(), createdAt.UTC().UnixMicro())
	return err
}
func (s *Store) CreateProject(ctx context.Context, p Project) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO projects(id,name,created_at,updated_at) VALUES(?,?,?,?)`, p.ID, p.Name, p.CreatedAt.UnixMicro(), p.UpdatedAt.UnixMicro())
	return err
}

func (s *Store) CreateWorkspaceForUser(ctx context.Context, p Project, member WorkspaceMember) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO projects(id,name,created_at,updated_at) VALUES(?,?,?,?)`, p.ID, p.Name, p.CreatedAt.UTC().UnixMicro(), p.UpdatedAt.UTC().UnixMicro()); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_members(workspace_id,user_id,role,created_at) VALUES(?,?,?,?)`, member.WorkspaceID, member.UserID, member.Role, member.CreatedAt.UTC().UnixMicro()); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) CreateProjectWithAPIKey(ctx context.Context, p Project, k APIKey) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO projects(id,name,created_at,updated_at) VALUES(?,?,?,?)`, p.ID, p.Name, p.CreatedAt.UTC().UnixMicro(), p.UpdatedAt.UTC().UnixMicro()); err != nil {
		return err
	}
	var exp any
	if k.ExpiresAt != nil {
		exp = k.ExpiresAt.UTC().UnixMicro()
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO api_keys(id,project_id,name,token_hash,role,expires_at,revoked) VALUES(?,?,?,?,?,?,?)`, k.ID, k.ProjectID, k.Name, k.TokenHash, k.Role, exp, k.Revoked); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ConsumeOAuthJTI(ctx context.Context, clientID, jti string, expiresAt, createdAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO oauth_jti(client_id,jti,expires_at,created_at) VALUES(?,?,?,?)`, clientID, jti, expiresAt.UTC().UnixMicro(), createdAt.UTC().UnixMicro())
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrAlreadyUsed
		}
		return err
	}
	return nil
}

func (s *Store) CreateUser(ctx context.Context, user User) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO users(id,email,name,password_hash,created_at) VALUES(?,?,?,?,?)`, user.ID, user.Email, user.Name, user.PasswordHash, user.CreatedAt.UTC().UnixMicro())
	return err
}

func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	var user User
	var created int64
	err := s.db.QueryRowContext(ctx, `SELECT id,email,name,password_hash,created_at FROM users WHERE email=?`, email).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return user, ErrNotFound
	}
	if err == nil {
		user.CreatedAt = time.UnixMicro(created).UTC()
	}
	return user, err
}

func (s *Store) UserByID(ctx context.Context, id string) (User, error) {
	var user User
	var created int64
	err := s.db.QueryRowContext(ctx, `SELECT id,email,name,password_hash,created_at FROM users WHERE id=?`, id).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return user, ErrNotFound
	}
	if err == nil {
		user.CreatedAt = time.UnixMicro(created).UTC()
	}
	return user, err
}

func (s *Store) AddWorkspaceMember(ctx context.Context, member WorkspaceMember) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO workspace_members(workspace_id,user_id,role,created_at) VALUES(?,?,?,?) ON CONFLICT(workspace_id,user_id) DO UPDATE SET role=excluded.role`, member.WorkspaceID, member.UserID, member.Role, member.CreatedAt.UTC().UnixMicro())
	return err
}

func (s *Store) CreateSession(ctx context.Context, tokenHash, userID, workspaceID string, expiresAt, createdAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO user_sessions(token_hash,user_id,workspace_id,expires_at,created_at) VALUES(?,?,?,?,?)`, tokenHash, userID, workspaceID, expiresAt.UTC().UnixMicro(), createdAt.UTC().UnixMicro())
	return err
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE token_hash=?`, tokenHash)
	return err
}

func (s *Store) AuthenticateUser(ctx context.Context, email, password string) (User, error) {
	user, err := s.UserByEmail(ctx, email)
	if err != nil {
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"), []byte(password))
		return User{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return User{}, ErrNotFound
	}
	return user, nil
}

func (s *Store) FirstWorkspaceForUser(ctx context.Context, userID string) (Project, error) {
	var project Project
	var created, updated int64
	err := s.db.QueryRowContext(ctx, `SELECT p.id,p.name,p.created_at,p.updated_at FROM projects p JOIN workspace_members m ON m.workspace_id=p.id WHERE m.user_id=? ORDER BY p.created_at,p.id LIMIT 1`, userID).Scan(&project.ID, &project.Name, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return project, ErrNotFound
	}
	if err == nil {
		project.CreatedAt = time.UnixMicro(created).UTC()
		project.UpdatedAt = time.UnixMicro(updated).UTC()
	}
	return project, err
}

func (s *Store) ListWorkspacesForUser(ctx context.Context, userID string) ([]Project, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT p.id,p.name,p.created_at,p.updated_at FROM projects p JOIN workspace_members m ON m.workspace_id=p.id WHERE m.user_id=? ORDER BY p.created_at,p.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	workspaces := make([]Project, 0)
	for rows.Next() {
		var workspace Project
		var created, updated int64
		if err := rows.Scan(&workspace.ID, &workspace.Name, &created, &updated); err != nil {
			return nil, err
		}
		workspace.CreatedAt = time.UnixMicro(created).UTC()
		workspace.UpdatedAt = time.UnixMicro(updated).UTC()
		workspaces = append(workspaces, workspace)
	}
	return workspaces, rows.Err()
}

func (s *Store) UserCanAccessWorkspace(ctx context.Context, userID, workspaceID string) error {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM workspace_members WHERE user_id=? AND workspace_id=?`, userID, workspaceID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *Store) SwitchSessionWorkspace(ctx context.Context, tokenHash, userID, workspaceID string) error {
	if err := s.UserCanAccessWorkspace(ctx, userID, workspaceID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `UPDATE user_sessions SET workspace_id=? WHERE token_hash=? AND user_id=?`, workspaceID, tokenHash, userID)
	return err
}
func (s *Store) CreateAPIKey(ctx context.Context, k APIKey) error {
	if k.Role == "" {
		k.Role = "project"
	}
	var exp any
	if k.ExpiresAt != nil {
		exp = k.ExpiresAt.UnixMicro()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO api_keys(id,project_id,name,token_hash,role,expires_at,revoked) VALUES(?,?,?,?,?,?,?)`, k.ID, k.ProjectID, k.Name, k.TokenHash, k.Role, exp, k.Revoked)
	return err
}
func (s *Store) Project(ctx context.Context, id string) (Project, error) {
	var p Project
	var c, u int64
	err := s.db.QueryRowContext(ctx, `SELECT id,name,created_at,updated_at FROM projects WHERE id=?`, id).Scan(&p.ID, &p.Name, &c, &u)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	if err == nil {
		p.CreatedAt = time.UnixMicro(c).UTC()
		p.UpdatedAt = time.UnixMicro(u).UTC()
	}
	return p, err
}
func (s *Store) Authenticate(ctx context.Context, token string) (APIKey, error) {
	var k APIKey
	var exp, last sql.NullInt64
	var revoked int
	err := s.db.QueryRowContext(ctx, `SELECT id,project_id,name,token_hash,role,expires_at,revoked,last_used_at FROM api_keys WHERE token_hash=?`, HashToken(token)).Scan(&k.ID, &k.ProjectID, &k.Name, &k.TokenHash, &k.Role, &exp, &revoked, &last)
	if errors.Is(err, sql.ErrNoRows) {
		var expires int64
		err = s.db.QueryRowContext(ctx, `SELECT project_id,expires_at FROM oauth_access_tokens WHERE token_hash=? AND expires_at>?`, HashToken(token), time.Now().UTC().UnixMicro()).Scan(&k.ProjectID, &expires)
		if errors.Is(err, sql.ErrNoRows) {
			var userID, workspaceID string
			err = s.db.QueryRowContext(ctx, `SELECT user_id,workspace_id FROM user_sessions WHERE token_hash=? AND expires_at>?`, HashToken(token), time.Now().UTC().UnixMicro()).Scan(&userID, &workspaceID)
			if errors.Is(err, sql.ErrNoRows) {
				return APIKey{}, ErrNotFound
			}
			if err != nil {
				return APIKey{}, err
			}
			k.ID = "user-session"
			k.UserID = userID
			k.ProjectID = workspaceID
			k.Name = "User session"
			k.Role = "member"
			return k, nil
		}
		if err != nil {
			return APIKey{}, err
		}
		k.ID = "oauth-access-token"
		k.Name = "OAuth access token"
		k.Role = "project"
		expiry := time.UnixMicro(expires).UTC()
		k.ExpiresAt = &expiry
		return k, nil
	}
	if err != nil {
		return k, err
	}
	k.Revoked = revoked != 0
	if exp.Valid {
		t := time.UnixMicro(exp.Int64).UTC()
		k.ExpiresAt = &t
	}
	if last.Valid {
		t := time.UnixMicro(last.Int64).UTC()
		k.LastUsedAt = &t
	}
	if k.Revoked || k.ExpiresAt != nil && k.ExpiresAt.Before(time.Now().UTC()) {
		return APIKey{}, ErrNotFound
	}
	now := time.Now().UTC()
	s.lastUsedMu.Lock()
	lastUsed, shouldUpdate := s.lastUsedMemory[k.ID], false
	if lastUsed.IsZero() || now.Sub(lastUsed) >= time.Minute {
		s.lastUsedMemory[k.ID] = now
		shouldUpdate = true
	}
	s.lastUsedMu.Unlock()
	if shouldUpdate {
		_, _ = s.db.ExecContext(ctx, `UPDATE api_keys SET last_used_at=? WHERE id=?`, now.UnixMicro(), k.ID)
	}
	return k, nil
}

func (s *Store) CleanupExpired(ctx context.Context, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE expires_at<=?; DELETE FROM oauth_access_tokens WHERE expires_at<=?; DELETE FROM oauth_jti WHERE expires_at<=?`, now.UTC().UnixMicro(), now.UTC().UnixMicro(), now.UTC().UnixMicro())
	return err
}

func (s *Store) EnsureAdmin(ctx context.Context, keyID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE api_keys SET role='admin' WHERE id=?`, keyID)
	return err
}

func (s *Store) ListProjects(ctx context.Context) ([]Project, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,created_at,updated_at FROM projects ORDER BY created_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Project
	for rows.Next() {
		var p Project
		var created, updated int64
		if err := rows.Scan(&p.ID, &p.Name, &created, &updated); err != nil {
			return nil, err
		}
		p.CreatedAt = time.UnixMicro(created).UTC()
		p.UpdatedAt = time.UnixMicro(updated).UTC()
		result = append(result, p)
	}
	return result, rows.Err()
}

func (s *Store) ListAPIKeys(ctx context.Context, projectID string) ([]APIKey, error) {
	query := `SELECT id,project_id,name,token_hash,role,expires_at,revoked,last_used_at FROM api_keys`
	args := []any{}
	if projectID != "" {
		query += ` WHERE project_id=?`
		args = append(args, projectID)
	}
	query += ` ORDER BY id`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []APIKey
	for rows.Next() {
		var k APIKey
		var exp, last sql.NullInt64
		var revoked int
		if err := rows.Scan(&k.ID, &k.ProjectID, &k.Name, &k.TokenHash, &k.Role, &exp, &revoked, &last); err != nil {
			return nil, err
		}
		k.Revoked = revoked != 0
		if exp.Valid {
			t := time.UnixMicro(exp.Int64).UTC()
			k.ExpiresAt = &t
		}
		if last.Valid {
			t := time.UnixMicro(last.Int64).UTC()
			k.LastUsedAt = &t
		}
		result = append(result, k)
	}
	return result, rows.Err()
}

func (s *Store) RevokeAPIKey(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE api_keys SET revoked=1 WHERE id=?`, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return ErrNotFound
	}
	return err
}
