package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lyra/lyra/internal/api"
	"github.com/lyra/lyra/internal/catalog"
	"github.com/lyra/lyra/internal/config"
	lyrapostgres "github.com/lyra/lyra/internal/persistence/postgres"
)

func Serve(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	var repo catalog.Repository = catalog.NewMemoryRepository()
	ready := func(*http.Request) error { return nil }
	if cfg.Database.URL != "" {
		pool, err := pgxpool.New(ctx, cfg.Database.URL)
		if err != nil {
			return fmt.Errorf("create postgres pool: %w", err)
		}
		defer pool.Close()
		repo = lyrapostgres.NewCatalog(pool)
		ready = func(r *http.Request) error { return pool.Ping(r.Context()) }
	}
	s := &http.Server{Addr: cfg.HTTP.Address, Handler: api.New(cfg, log, repo, ready), ReadTimeout: cfg.HTTP.ReadTimeout, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: cfg.HTTP.WriteTimeout, IdleTimeout: cfg.HTTP.IdleTimeout}
	go func() { <-ctx.Done(); s.Shutdown(context.Background()) }()
	err := s.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
