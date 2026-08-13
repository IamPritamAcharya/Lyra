package postgres

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lyra/lyra/internal/catalog"
)

type CatalogRepository struct{ pool *pgxpool.Pool }

func NewCatalog(pool *pgxpool.Pool) *CatalogRepository { return &CatalogRepository{pool} }
func (r *CatalogRepository) Create(ctx context.Context, in catalog.CreateTrack) (catalog.Track, error) {
	return scanTrack(r.pool.QueryRow(ctx, `INSERT INTO tracks (public_id,title,artist_name,album_name,status) VALUES ($1,$2,$3,$4,'CREATED') RETURNING id,public_id::text,title,artist_name,album_name,status,created_at,updated_at`, newUUID(), in.Title, in.ArtistName, in.AlbumName))
}
func (r *CatalogRepository) Get(ctx context.Context, id string) (catalog.Track, error) {
	return scanTrack(r.pool.QueryRow(ctx, `SELECT id,public_id::text,title,artist_name,album_name,status,created_at,updated_at FROM tracks WHERE public_id=$1 AND deleted_at IS NULL`, id))
}
func (r *CatalogRepository) GetByID(ctx context.Context, id int64) (catalog.Track, error) {
	return scanTrack(r.pool.QueryRow(ctx, `SELECT id,public_id::text,title,artist_name,album_name,status,created_at,updated_at FROM tracks WHERE id=$1 AND deleted_at IS NULL`, id))
}
func (r *CatalogRepository) List(ctx context.Context, limit, offset int) ([]catalog.Track, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,public_id::text,title,artist_name,album_name,status,created_at,updated_at FROM tracks WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []catalog.Track
	for rows.Next() {
		t, e := scanTrack(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
func (r *CatalogRepository) Transition(ctx context.Context, id string, to catalog.Status, reason *string) (catalog.Track, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return catalog.Track{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	cur, err := scanTrack(tx.QueryRow(ctx, `SELECT id,public_id::text,title,artist_name,album_name,status,created_at,updated_at FROM tracks WHERE public_id=$1 FOR UPDATE`, id))
	if err != nil {
		return catalog.Track{}, err
	}
	if err := catalog.ValidateTransition(cur.Status, to); err != nil {
		return catalog.Track{}, err
	}
	out, err := scanTrack(tx.QueryRow(ctx, `UPDATE tracks SET status=$2,failure_reason=$3,updated_at=now(),deleted_at=CASE WHEN $2='DELETED' THEN now() ELSE deleted_at END WHERE public_id=$1 RETURNING id,public_id::text,title,artist_name,album_name,status,created_at,updated_at`, id, to, reason))
	if err != nil {
		return catalog.Track{}, err
	}
	return out, tx.Commit(ctx)
}

type row interface{ Scan(...any) error }

func scanTrack(row row) (catalog.Track, error) {
	var t catalog.Track
	err := row.Scan(&t.ID, &t.PublicID, &t.Title, &t.ArtistName, &t.AlbumName, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, catalog.ErrTrackNotFound
	}
	if err != nil {
		return t, fmt.Errorf("scan track: %w", err)
	}
	return t, nil
}
func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("cryptographic random unavailable")
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}
