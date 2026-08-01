// Package catalog owns reference-track metadata and its explicit lifecycle.
package catalog

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrTrackNotFound     = errors.New("track not found")
	ErrInvalidTransition = errors.New("invalid track status transition")
)

type Status string

const (
	Created    Status = "CREATED"
	Uploaded   Status = "UPLOADED"
	Indexing   Status = "INDEXING"
	Ready      Status = "READY"
	Failed     Status = "FAILED"
	Reindexing Status = "REINDEXING"
	Deleting   Status = "DELETING"
	Deleted    Status = "DELETED"
)

type Track struct {
	ID                   int64
	PublicID             string
	Title, ArtistName    string
	AlbumName            *string
	Status               Status
	FailureReason        *string
	FingerprintVersion   *int16
	CreatedAt, UpdatedAt time.Time
	DeletedAt            *time.Time
}
type CreateTrack struct {
	Title, ArtistName string
	AlbumName         *string
}
type Repository interface {
	Create(context.Context, CreateTrack) (Track, error)
	Get(context.Context, string) (Track, error)
	List(context.Context, int, int) ([]Track, error)
	Transition(context.Context, string, Status, *string) (Track, error)
}

func CanTransition(from, to Status) bool {
	return map[Status]map[Status]bool{Created: {Uploaded: true, Deleting: true}, Uploaded: {Indexing: true, Deleting: true}, Indexing: {Ready: true, Failed: true}, Ready: {Reindexing: true, Deleting: true}, Failed: {Indexing: true, Deleting: true}, Reindexing: {Ready: true, Failed: true}, Deleting: {Deleted: true}}[from][to]
}
func ValidateTransition(from, to Status) error {
	if !CanTransition(from, to) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, from, to)
	}
	return nil
}
