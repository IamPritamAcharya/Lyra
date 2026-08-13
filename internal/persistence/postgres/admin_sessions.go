package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionStore struct {
	pool *pgxpool.Pool
}

func NewSessionStore(pool *pgxpool.Pool) *SessionStore { return &SessionStore{pool: pool} }

func (r *SessionStore) Create(ctx context.Context, tokenHash, csrfHash []byte, expires time.Time) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM admin_sessions WHERE expires_at <= now()`); err != nil {
		return fmt.Errorf("remove expired admin sessions: %w", err)
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO admin_sessions(token_hash,csrf_token_hash,expires_at) VALUES($1,$2,$3)`, tokenHash, csrfHash, expires)
	if err != nil {
		return fmt.Errorf("create admin session: %w", err)
	}
	return nil
}

func (r *SessionStore) Get(ctx context.Context, tokenHash []byte) (time.Time, []byte, error) {
	var expires time.Time
	var csrf []byte
	err := r.pool.QueryRow(ctx, `SELECT expires_at,csrf_token_hash FROM admin_sessions WHERE token_hash=$1`, tokenHash).Scan(&expires, &csrf)
	if err != nil {
		return time.Time{}, nil, fmt.Errorf("get admin session: %w", err)
	}
	return expires, csrf, nil
}

func (r *SessionStore) Delete(ctx context.Context, tokenHash []byte) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM admin_sessions WHERE token_hash=$1`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete admin session: %w", err)
	}
	return nil
}
