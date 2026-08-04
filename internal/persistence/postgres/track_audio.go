package postgres

import (
	"context"
	"fmt"
	"github.com/lyra/lyra/internal/catalog"
)

type AudioMetadata struct {
	ObjectKey, SHA256, MimeType, OriginalFilename string
	SizeBytes                                     int64
}

func (r *CatalogRepository) RecordAudio(ctx context.Context, publicID string, m AudioMetadata) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO track_audio(track_id,object_key,sha256,size_bytes,mime_type,original_filename) SELECT id,$2,$3,$4,$5,$6 FROM tracks WHERE public_id=$1 ON CONFLICT (track_id) DO UPDATE SET object_key=EXCLUDED.object_key,sha256=EXCLUDED.sha256,size_bytes=EXCLUDED.size_bytes,mime_type=EXCLUDED.mime_type,original_filename=EXCLUDED.original_filename`, publicID, m.ObjectKey, m.SHA256, m.SizeBytes, m.MimeType, m.OriginalFilename)
	if err != nil {
		return fmt.Errorf("record track audio: %w", err)
	}
	return nil
}
func (r *CatalogRepository) ObjectKey(ctx context.Context, publicID string) (string, error) {
	var key string
	err := r.pool.QueryRow(ctx, `SELECT a.object_key FROM track_audio a JOIN tracks t ON t.id=a.track_id WHERE t.public_id=$1`, publicID).Scan(&key)
	if err != nil {
		return "", fmt.Errorf("lookup object key: %w", err)
	}
	return key, nil
}
func (r *CatalogRepository) Fail(ctx context.Context, publicID, reason string) error {
	track, err := r.Get(ctx, publicID)
	if err != nil {
		return err
	}
	if track.Status == catalog.Uploaded {
		if _, err = r.Transition(ctx, publicID, catalog.Indexing, nil); err != nil {
			return err
		}
	}
	_, err = r.Transition(ctx, publicID, catalog.Failed, &reason)
	return err
}
