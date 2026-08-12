package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/lyra/lyra/internal/app"
	"github.com/lyra/lyra/internal/config"
	"github.com/lyra/lyra/internal/fingerprint"
	lyrapostgres "github.com/lyra/lyra/internal/persistence/postgres"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "eval":
		if len(os.Args) != 4 || os.Args[2] != "--manifest" {
			usage()
		}
		cfg, err := config.Load()
		if err != nil {
			fail(err)
		}
		report, err := app.Evaluate(context.Background(), cfg, os.Args[3])
		if err != nil {
			fail(err)
		}
		if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
			fail(err)
		}
	case "migrate":
		cfg, err := config.Load()
		if err != nil {
			fail(err)
		}
		if cfg.Database.URL == "" {
			fail(fmt.Errorf("DATABASE_URL is required for migrations"))
		}
		if err := lyrapostgres.MigrateUp(cfg.Database.URL, "db/migrations"); err != nil {
			fail(err)
		}
	case "worker":
		cfg, err := config.Load()
		if err != nil {
			fail(err)
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := app.Worker(ctx, cfg); err != nil {
			fail(err)
		}
	case "serve":
		cfg, err := config.Load()
		if err != nil {
			fail(err)
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := app.Serve(ctx, cfg, slog.Default()); err != nil {
			fail(err)
		}
	case "fingerprint":
		if len(os.Args) != 3 {
			usage()
		}
		samples, err := fingerprint.ReadCanonicalWAV(os.Args[2])
		if err != nil {
			fail(err)
		}
		fps, err := fingerprint.Extract(samples)
		if errors.Is(err, fingerprint.ErrInsufficientSignal) {
			fail(err)
		}
		if err != nil {
			fail(fmt.Errorf("extract fingerprints: %w", err))
		}
		if err := json.NewEncoder(os.Stdout).Encode(fps); err != nil {
			fail(err)
		}
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: lyra serve | worker | migrate | eval --manifest <path> | fingerprint <canonical-11025Hz-mono-pcm16.wav>")
	os.Exit(2)
}
func fail(err error) { fmt.Fprintln(os.Stderr, "lyra:", err); os.Exit(1) }
