package app

import (
	"context"
	"github.com/lyra/lyra/internal/api"
	"github.com/lyra/lyra/internal/config"
	"log/slog"
	"net/http"
	"time"
)

func Serve(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	s := &http.Server{Addr: cfg.HTTP.Address, Handler: api.New(cfg, log), ReadTimeout: cfg.HTTP.ReadTimeout, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: cfg.HTTP.WriteTimeout, IdleTimeout: cfg.HTTP.IdleTimeout}
	go func() { <-ctx.Done(); s.Shutdown(context.Background()) }()
	err := s.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
