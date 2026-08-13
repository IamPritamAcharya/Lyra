package queue

import (
	"context"
	"fmt"
	"github.com/hibiken/asynq"
	"log/slog"
)

// RunWorker owns the worker process lifecycle; callers provide the bounded task set.
func RunWorker(ctx context.Context, redisAddr string, handler asynq.Handler, log *slog.Logger) error {
	server := asynq.NewServer(asynq.RedisClientOpt{Addr: redisAddr}, asynq.Config{Concurrency: 1, Queues: map[string]int{"default": 1}})
	mux := asynq.NewServeMux()
	mux.Handle(TypeFingerprintTrack, handler)
	done := make(chan error, 1)
	go func() { done <- server.Run(mux) }()
	log.Info("worker_started", "queue", "default", "concurrency", 1)
	select {
	case err := <-done:
		log.Error("worker_stopped_unexpectedly", "error", err)
		return fmt.Errorf("run worker: %w", err)
	case <-ctx.Done():
		log.Info("worker_shutdown_started")
		server.Shutdown()
		err := <-done
		if err != nil {
			return err
		}
		log.Info("worker_shutdown_completed")
		return nil
	}
}
