package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/lyra/lyra/internal/catalog"
	"github.com/lyra/lyra/internal/fingerprint"
)

// StoreTrack replaces one version atomically. A failed copy leaves the prior
// indexed version intact, and repeated jobs converge on the same final state.
func (r *CatalogRepository) StoreTrack(ctx context.Context, publicID string, version int16, fps []fingerprint.Fingerprint) error {
	if len(fps) == 0 {
		return fingerprint.ErrInsufficientSignal
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var id int64
	var status catalog.Status
	if err := tx.QueryRow(ctx, `SELECT id,status FROM tracks WHERE public_id=$1 FOR UPDATE`, publicID).Scan(&id, &status); err != nil {
		return fmt.Errorf("lock track: %w", err)
	}
	next := catalog.Indexing
	if status == catalog.Ready {
		next = catalog.Reindexing
	}
	if err := catalog.ValidateTransition(status, next); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE tracks SET status=$2,updated_at=now(),failure_reason=NULL WHERE id=$1`, id, next); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM fingerprints WHERE track_id=$1 AND algorithm_version=$2`, id, version); err != nil {
		return err
	}
	seen := map[[2]uint32]struct{}{}
	rows := make([][]any, 0, len(fps))
	for _, fp := range fps {
		k := [2]uint32{fp.Hash, uint32(fp.AnchorFrame)}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		rows = append(rows, []any{version, int32(fp.Hash), id, fp.AnchorFrame})
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"fingerprints"}, []string{"algorithm_version", "hash", "track_id", "anchor_frame"}, pgx.CopyFromRows(rows)); err != nil {
		return fmt.Errorf("bulk insert fingerprints: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE tracks SET status='READY',fingerprint_version=$2,updated_at=now() WHERE id=$1`, id, version); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
