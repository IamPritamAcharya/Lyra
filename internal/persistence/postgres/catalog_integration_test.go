//go:build integration

package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lyra/lyra/internal/catalog"
)

func TestCatalogRepositoryLifecycle(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is required")
	}
	if err := MigrateUp(url, "../../../db/migrations"); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repo := NewCatalog(pool)
	track, err := repo.Create(context.Background(), catalog.CreateTrack{Title: "integration", ArtistName: "lyra"})
	if err != nil {
		t.Fatal(err)
	}
	for _, next := range []catalog.Status{catalog.Uploaded, catalog.Indexing, catalog.Ready, catalog.Deleting, catalog.Deleted} {
		track, err = repo.Transition(context.Background(), track.PublicID, next, nil)
		if err != nil {
			t.Fatalf("transition %s: %v", next, err)
		}
	}
	if track.Status != catalog.Deleted {
		t.Fatalf("status = %s", track.Status)
	}
	if _, err := repo.Get(context.Background(), track.PublicID); err != catalog.ErrTrackNotFound {
		t.Fatalf("deleted fetch error = %v", err)
	}
}
