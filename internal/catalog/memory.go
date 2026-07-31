package catalog

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// MemoryRepository is only a local development fallback; PostgreSQL is the production source of truth.
type MemoryRepository struct {
	mu     sync.Mutex
	next   int64
	tracks map[string]Track
}

func NewMemoryRepository() *MemoryRepository { return &MemoryRepository{tracks: map[string]Track{}} }
func (r *MemoryRepository) Create(_ context.Context, in CreateTrack) (Track, error) {
	if in.Title == "" || in.ArtistName == "" {
		return Track{}, fmt.Errorf("title and artist are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	now := time.Now().UTC()
	t := Track{ID: r.next, PublicID: newID(), Title: in.Title, ArtistName: in.ArtistName, AlbumName: in.AlbumName, Status: Created, CreatedAt: now, UpdatedAt: now}
	r.tracks[t.PublicID] = t
	return t, nil
}
func (r *MemoryRepository) Get(_ context.Context, id string) (Track, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tracks[id]
	if !ok || t.Status == Deleted {
		return Track{}, ErrTrackNotFound
	}
	return t, nil
}
func (r *MemoryRepository) List(_ context.Context, limit, offset int) ([]Track, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Track, 0)
	for _, t := range r.tracks {
		if t.Status != Deleted {
			out = append(out, t)
		}
	}
	if offset >= len(out) {
		return out[:0], nil
	}
	if limit <= 0 || limit > len(out)-offset {
		limit = len(out) - offset
	}
	return out[offset : offset+limit], nil
}
func (r *MemoryRepository) Transition(ctx context.Context, id string, to Status, reason *string) (Track, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tracks[id]
	if !ok {
		return Track{}, ErrTrackNotFound
	}
	if err := ValidateTransition(t.Status, to); err != nil {
		return Track{}, err
	}
	t.Status = to
	t.FailureReason = reason
	t.UpdatedAt = time.Now().UTC()
	if to == Deleted {
		v := t.UpdatedAt
		t.DeletedAt = &v
	}
	r.tracks[id] = t
	return t, nil
}
func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}
