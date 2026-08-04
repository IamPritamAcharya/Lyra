// Package ingest coordinates private reference uploads and asynchronous indexing.
package ingest

import (
	"context"
	"fmt"
	"github.com/lyra/lyra/internal/catalog"
	"github.com/lyra/lyra/internal/objectstore"
	"io"
)

type Enqueuer interface{ Enqueue(string) error }
type Service struct {
	Catalog catalog.Repository
	Objects objectstore.ObjectStore
	Queue   Enqueuer
}

// Upload stores only reference audio, then makes the lifecycle state visible before enqueueing work.
func (s Service) Upload(ctx context.Context, trackID, key string, body io.Reader, size int64, mime string) error {
	if size <= 0 {
		return fmt.Errorf("empty upload")
	}
	if err := s.Objects.Put(ctx, key, body, size, mime); err != nil {
		return fmt.Errorf("store reference: %w", err)
	}
	if _, err := s.Catalog.Transition(ctx, trackID, catalog.Uploaded, nil); err != nil {
		return fmt.Errorf("mark uploaded: %w", err)
	}
	if err := s.Queue.Enqueue(trackID); err != nil {
		return fmt.Errorf("enqueue fingerprint task: %w", err)
	}
	return nil
}
