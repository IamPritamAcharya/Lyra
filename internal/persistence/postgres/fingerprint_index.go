package postgres

import (
	"context"
	"fmt"

	"github.com/lyra/lyra/internal/identify"
)

// Lookup batches every distinct query hash into one PostgreSQL request.
func (r *CatalogRepository) Lookup(ctx context.Context, version int16, hashes []uint32) ([]identify.Posting, error) {
	if len(hashes) == 0 {
		return nil, nil
	}
	values := make([]int32, len(hashes))
	for i, hash := range hashes {
		values[i] = int32(hash)
	}
	rows, err := r.pool.Query(ctx, `SELECT hash, track_id, anchor_frame FROM fingerprints WHERE algorithm_version=$1 AND hash = ANY($2)`, version, values)
	if err != nil {
		return nil, fmt.Errorf("lookup fingerprints: %w", err)
	}
	defer rows.Close()
	postings := make([]identify.Posting, 0)
	for rows.Next() {
		var hash int32
		var p identify.Posting
		if err := rows.Scan(&hash, &p.TrackID, &p.AnchorFrame); err != nil {
			return nil, fmt.Errorf("scan posting: %w", err)
		}
		p.Hash = uint32(hash)
		postings = append(postings, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate postings: %w", err)
	}
	return postings, nil
}

// FilterHashes removes very common hashes using maintained posting statistics.
func (r *CatalogRepository) FilterHashes(ctx context.Context, version int16, hashes []uint32, maximum int64) ([]uint32, error) {
	if len(hashes) == 0 {
		return nil, nil
	}
	values := make([]int32, len(hashes))
	for i, h := range hashes {
		values[i] = int32(h)
	}
	rows, err := r.pool.Query(ctx, `SELECT hash FROM fingerprint_hash_stats WHERE algorithm_version=$1 AND hash=ANY($2) AND posting_count <= $3 ORDER BY hash`, version, values, maximum)
	if err != nil {
		return nil, fmt.Errorf("filter common hashes: %w", err)
	}
	defer rows.Close()
	filtered := make([]uint32, 0, len(hashes))
	for rows.Next() {
		var hash int32
		if err := rows.Scan(&hash); err != nil {
			return nil, err
		}
		filtered = append(filtered, uint32(hash))
	}
	return filtered, rows.Err()
}
