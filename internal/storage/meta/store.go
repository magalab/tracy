package meta

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type Project struct {
	ID, Name             string
	CreatedAt, UpdatedAt time.Time
}
type APIKey struct {
	ID, ProjectID, Name, TokenHash string
	ExpiresAt                      *time.Time
	Revoked                        bool
	LastUsedAt                     *time.Time
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Store) Migrate(ctx context.Context) error { return migrate(ctx, s.db) }
func migrate(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS projects (id TEXT PRIMARY KEY, name TEXT NOT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL); CREATE TABLE IF NOT EXISTS api_keys (id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), name TEXT NOT NULL, token_hash TEXT NOT NULL UNIQUE, expires_at INTEGER, revoked INTEGER NOT NULL DEFAULT 0, last_used_at INTEGER); CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(token_hash);`)
	return err
}
func (s *Store) CreateProject(ctx context.Context, p Project) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO projects(id,name,created_at,updated_at) VALUES(?,?,?,?)`, p.ID, p.Name, p.CreatedAt.UnixMicro(), p.UpdatedAt.UnixMicro())
	return err
}
func (s *Store) CreateAPIKey(ctx context.Context, k APIKey) error {
	var exp any
	if k.ExpiresAt != nil {
		exp = k.ExpiresAt.UnixMicro()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO api_keys(id,project_id,name,token_hash,expires_at,revoked) VALUES(?,?,?,?,?,?)`, k.ID, k.ProjectID, k.Name, k.TokenHash, exp, k.Revoked)
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
	err := s.db.QueryRowContext(ctx, `SELECT id,project_id,name,token_hash,expires_at,revoked,last_used_at FROM api_keys WHERE token_hash=?`, HashToken(token)).Scan(&k.ID, &k.ProjectID, &k.Name, &k.TokenHash, &exp, &revoked, &last)
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
