package meta

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/panda/tracy/internal/annotation"
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
	ExpiresAt                            *time.Time
	Revoked                              bool
	LastUsedAt                           *time.Time
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Store) Migrate(ctx context.Context) error { return migrate(ctx, s.db) }
func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS projects (id TEXT PRIMARY KEY, name TEXT NOT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL); CREATE TABLE IF NOT EXISTS api_keys (id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), name TEXT NOT NULL, token_hash TEXT NOT NULL UNIQUE, role TEXT NOT NULL DEFAULT 'project', expires_at INTEGER, revoked INTEGER NOT NULL DEFAULT 0, last_used_at INTEGER); CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(token_hash);`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS annotations (id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), trace_id TEXT NOT NULL, span_id TEXT NOT NULL DEFAULT '', annotation_key TEXT NOT NULL, score REAL, label TEXT NOT NULL DEFAULT '', comment TEXT NOT NULL DEFAULT '', created_by TEXT NOT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL); CREATE INDEX IF NOT EXISTS idx_annotations_project_trace ON annotations(project_id,trace_id,created_at);`); err != nil {
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
func (s *Store) CreateProject(ctx context.Context, p Project) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO projects(id,name,created_at,updated_at) VALUES(?,?,?,?)`, p.ID, p.Name, p.CreatedAt.UnixMicro(), p.UpdatedAt.UnixMicro())
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
		return k, ErrNotFound
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
	_, _ = s.db.ExecContext(ctx, `UPDATE api_keys SET last_used_at=? WHERE id=?`, time.Now().UTC().UnixMicro(), k.ID)
	return k, nil
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

func (s *Store) CreateAnnotation(ctx context.Context, item annotation.Annotation) error {
	var score any
	if item.Score != nil {
		score = *item.Score
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO annotations(id,project_id,trace_id,span_id,annotation_key,score,label,comment,created_by,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, item.ID, item.ProjectID, item.TraceID, item.SpanID, item.Key, score, item.Label, item.Comment, item.CreatedBy, item.CreatedAt.UTC().UnixMicro(), item.UpdatedAt.UTC().UnixMicro())
	return err
}

func (s *Store) ListAnnotations(ctx context.Context, projectID, traceID string) ([]annotation.Annotation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,project_id,trace_id,span_id,annotation_key,score,label,comment,created_by,created_at,updated_at FROM annotations WHERE project_id=? AND trace_id=? ORDER BY created_at,id`, projectID, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]annotation.Annotation, 0)
	for rows.Next() {
		var item annotation.Annotation
		var score sql.NullFloat64
		var created, updated int64
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.TraceID, &item.SpanID, &item.Key, &score, &item.Label, &item.Comment, &item.CreatedBy, &created, &updated); err != nil {
			return nil, err
		}
		if score.Valid {
			v := score.Float64
			item.Score = &v
		}
		item.CreatedAt = time.UnixMicro(created).UTC()
		item.UpdatedAt = time.UnixMicro(updated).UTC()
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) DeleteAnnotation(ctx context.Context, projectID, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM annotations WHERE project_id=? AND id=?`, projectID, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return ErrNotFound
	}
	return err
}
