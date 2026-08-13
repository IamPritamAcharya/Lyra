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
	"github.com/lyra/lyra/internal/auth"
	"github.com/lyra/lyra/internal/config"
	"github.com/lyra/lyra/internal/identify"
	"github.com/lyra/lyra/internal/ingest"
	"github.com/lyra/lyra/internal/objectstore"
	lyrapostgres "github.com/lyra/lyra/internal/persistence/postgres"
	"github.com/lyra/lyra/internal/queue"
)

func Serve(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	log.Info("server_starting", "address", cfg.HTTP.Address, "log_format", cfg.Observability.LogFormat, "log_level", cfg.Observability.LogLevel)
	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("create postgres pool: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	log.Info("dependency_ready", "dependency", "postgres")
	repo := lyrapostgres.NewCatalog(pool)
	identifier := identify.FileIdentifier{Audio: audio.NewProcessor(), Matcher: identify.New(repo, identify.DefaultConfig())}
	objects, err := objectstore.NewS3(objectstore.Config{Endpoint: cfg.Storage.Endpoint, AccessKey: cfg.Storage.AccessKey, SecretKey: cfg.Storage.SecretKey, Bucket: cfg.Storage.Bucket, Secure: cfg.Storage.Secure})
	if err != nil {
		return err
	}
	if err := objects.EnsureBucket(ctx); err != nil {
		return fmt.Errorf("ensure reference bucket: %w", err)
	}
	log.Info("dependency_ready", "dependency", "object_storage")
	client := queue.NewClient(cfg.Redis.Address)
	defer client.Close()
	log.Info("dependency_configured", "dependency", "valkey")
	uploader := ingest.Service{Catalog: repo, Objects: objects, Queue: client, Audio: repo, Log: log}
	ready := func(r *http.Request) error { return pool.Ping(r.Context()) }
	adminAuth := auth.New(cfg.Security.AdminUsername, cfg.Security.AdminPasswordHash, lyrapostgres.NewSessionStore(pool))
	s := &http.Server{Addr: cfg.HTTP.Address, Handler: api.New(cfg, log, repo, ready, identifier, uploader, adminAuth), ReadTimeout: cfg.HTTP.ReadTimeout, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: cfg.HTTP.WriteTimeout, IdleTimeout: cfg.HTTP.IdleTimeout}
	go func() {
		<-ctx.Done()
		log.Info("server_shutdown_started")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.Shutdown(shutdownCtx); err != nil {
			log.Error("server_shutdown_failed", "error", err)
			return
		}
		log.Info("server_shutdown_completed")
	}()
	log.Info("server_started", "address", cfg.HTTP.Address)
	err = s.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func Worker(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	log.Info("worker_starting", "queue", "default", "concurrency", 1)
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
	worker := ingest.Worker{Objects: objects, Locate: repo, Store: repo, Failures: repo, Audio: audio.NewProcessor(), Log: log}
	return queue.RunWorker(ctx, cfg.Redis.Address, asynq.HandlerFunc(worker.HandleFingerprintTask), log)
}
