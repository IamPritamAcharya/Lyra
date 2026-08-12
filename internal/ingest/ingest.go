// Package ingest coordinates private reference uploads and asynchronous indexing.
package ingest

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/lyra/lyra/internal/catalog"
	"github.com/lyra/lyra/internal/objectstore"
	"io"
)

type Enqueuer interface{ Enqueue(string) error }
type AudioMetadata struct {
	ObjectKey, SHA256, MimeType, OriginalFilename string
	SizeBytes                                     int64
}
type AudioRecorder interface {
	RecordAudio(context.Context, string, AudioMetadata) error
}
type Service struct {
	Catalog catalog.Repository
	Objects objectstore.ObjectStore
	Queue   Enqueuer
	Audio   AudioRecorder
}

// Upload stores only reference audio, then makes the lifecycle state visible before enqueueing work.
func (s Service) Upload(ctx context.Context, trackID, key, filename string, body io.Reader, size int64, mime string) error {
	if size <= 0 {
		return fmt.Errorf("empty upload")
	}
	h := sha256.New()
	if err := s.Objects.Put(ctx, key, io.TeeReader(body, h), size, mime); err != nil {
		return fmt.Errorf("store reference: %w", err)
	}
	if err := s.Audio.RecordAudio(ctx, trackID, AudioMetadata{ObjectKey: key, SHA256: fmt.Sprintf("%x", h.Sum(nil)), MimeType: mime, OriginalFilename: filename, SizeBytes: size}); err != nil {
		return fmt.Errorf("record reference audio: %w", err)
	}
	if _, err := s.Catalog.Transition(ctx, trackID, catalog.Uploaded, nil); err != nil {
		return fmt.Errorf("mark uploaded: %w", err)
	}
	if err := s.Queue.Enqueue(trackID); err != nil {
		return fmt.Errorf("enqueue fingerprint task: %w", err)
	}
	return nil
}
