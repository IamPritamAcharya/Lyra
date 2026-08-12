package app

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lyra/lyra/internal/audio"
	"github.com/lyra/lyra/internal/config"
	lyraeval "github.com/lyra/lyra/internal/eval"
	"github.com/lyra/lyra/internal/identify"
	lyrapostgres "github.com/lyra/lyra/internal/persistence/postgres"
)

func Evaluate(ctx context.Context, cfg config.Config, manifestPath string) (lyraeval.Report, error) {
	if cfg.Database.URL == "" {
		return lyraeval.Report{}, fmt.Errorf("DATABASE_URL is required for eval")
	}
	manifest, err := lyraeval.Load(manifestPath)
	if err != nil {
		return lyraeval.Report{}, err
	}
	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		return lyraeval.Report{}, err
	}
	defer pool.Close()
	repo := lyrapostgres.NewCatalog(pool)
	id := identify.FileIdentifier{Audio: audio.NewProcessor(), Matcher: identify.New(repo, identify.DefaultConfig())}
	return lyraeval.Run(ctx, manifest, evalAdapter{id: id, repo: repo})
}

type evalAdapter struct {
	id   identify.FileIdentifier
	repo *lyrapostgres.CatalogRepository
}

func (a evalAdapter) Identify(ctx context.Context, path string) (lyraeval.Result, error) {
	got, err := a.id.IdentifyFile(ctx, path)
	if err != nil && err != identify.ErrNoMatch {
		return lyraeval.Result{}, err
	}
	if !got.Matched {
		return lyraeval.Result{}, nil
	}
	track, err := a.repo.GetByID(ctx, got.Candidate.TrackID)
	if err != nil {
		return lyraeval.Result{}, err
	}
	return lyraeval.Result{Matched: true, TrackID: track.PublicID}, nil
}
