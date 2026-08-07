package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lyra/lyra/internal/api"
	"github.com/lyra/lyra/internal/audio"
	"github.com/lyra/lyra/internal/catalog"
	"github.com/lyra/lyra/internal/config"
	"github.com/lyra/lyra/internal/identify"
	"github.com/lyra/lyra/internal/ingest"
	"github.com/lyra/lyra/internal/objectstore"
	lyrapostgres "github.com/lyra/lyra/internal/persistence/postgres"
	"github.com/lyra/lyra/internal/queue"
)

func Serve(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	var repo catalog.Repository = catalog.NewMemoryRepository()
	ready := func(*http.Request) error { return nil }
	var identifier api.FileIdentifier
	if cfg.Database.URL != "" {
		pool, err := pgxpool.New(ctx, cfg.Database.URL)
		if err != nil {
			return fmt.Errorf("create postgres pool: %w", err)
		}
		defer pool.Close()
		repo = lyrapostgres.NewCatalog(pool)
		identifier = identify.FileIdentifier{Audio: audio.NewProcessor(), Matcher: identify.New(lyrapostgres.NewCatalog(pool), identify.DefaultConfig())}
		ready = func(r *http.Request) error { return pool.Ping(r.Context()) }
	}
	s := &http.Server{Addr: cfg.HTTP.Address, Handler: api.New(cfg, log, repo, ready, identifier), ReadTimeout: cfg.HTTP.ReadTimeout, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: cfg.HTTP.WriteTimeout, IdleTimeout: cfg.HTTP.IdleTimeout}
	go func() { <-ctx.Done(); s.Shutdown(context.Background()) }()
	err := s.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func Worker(ctx context.Context, cfg config.Config) error {
	if cfg.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required for worker")
	}
	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		return err
	}
	defer pool.Close()
	objects, err := objectstore.NewS3(objectstore.Config{Endpoint: cfg.Storage.Endpoint, AccessKey: cfg.Storage.AccessKey, SecretKey: cfg.Storage.SecretKey, Bucket: cfg.Storage.Bucket, Secure: cfg.Storage.Secure})
	if err != nil {
		return err
	}
	repo := lyrapostgres.NewCatalog(pool)
	worker := ingest.Worker{Objects: objects, Locate: repo, Store: repo, Failures: repo, Audio: audio.NewProcessor()}
	return queue.RunWorker(ctx, cfg.Redis.Address, asynq.HandlerFunc(worker.HandleFingerprintTask))
}
