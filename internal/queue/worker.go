package queue

import (
	"context"
	"fmt"
	"github.com/hibiken/asynq"
)

// RunWorker owns the worker process lifecycle; callers provide the bounded task set.
func RunWorker(ctx context.Context, redisAddr string, handler asynq.Handler) error {
	server := asynq.NewServer(asynq.RedisClientOpt{Addr: redisAddr}, asynq.Config{Concurrency: 1, Queues: map[string]int{"default": 1}})
	mux := asynq.NewServeMux()
	mux.Handle(TypeFingerprintTrack, handler)
	done := make(chan error, 1)
	go func() { done <- server.Run(mux) }()
	select {
	case err := <-done:
		return fmt.Errorf("run worker: %w", err)
	case <-ctx.Done():
		server.Shutdown()
		return <-done
	}
}
