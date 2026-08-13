package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/hibiken/asynq"
	"github.com/lyra/lyra/internal/audio"
	"github.com/lyra/lyra/internal/fingerprint"
	"github.com/lyra/lyra/internal/queue"
)

type ReferenceLocator interface {
	ObjectKey(context.Context, string) (string, error)
}
type FingerprintStore interface {
	StoreTrack(context.Context, string, int16, []fingerprint.Fingerprint) error
}
type FailureRecorder interface {
	Fail(context.Context, string, string) error
}
type Worker struct {
	Objects interface {
		Get(context.Context, string) (io.ReadCloser, error)
	}
	Locate   ReferenceLocator
	Store    FingerprintStore
	Failures FailureRecorder
	Audio    audio.Processor
	Log      *slog.Logger
}

func (w Worker) HandleFingerprintTask(ctx context.Context, task *asynq.Task) error {
	started := time.Now()
	var payload queue.FingerprintTrackPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		w.Log.Error("fingerprint_job_failed", "reason", "invalid_payload", "error", err)
		return fmt.Errorf("decode fingerprint task: %w", err)
	}
	w.Log.Info("fingerprint_job_started", "track_id", payload.TrackID)
	fingerprintCount, err := w.index(ctx, payload.TrackID)
	if err != nil && w.Failures != nil {
		_ = w.Failures.Fail(ctx, payload.TrackID, err.Error())
		w.Log.Error("fingerprint_job_failed", "track_id", payload.TrackID, "duration_ms", time.Since(started).Milliseconds(), "error", err)
	} else if err == nil {
		w.Log.Info("track_indexed", "track_id", payload.TrackID, "status", "READY", "fingerprints", fingerprintCount, "duration_ms", time.Since(started).Milliseconds())
	}
	return err
}
func (w Worker) index(ctx context.Context, trackID string) (int, error) {
	key, err := w.Locate.ObjectKey(ctx, trackID)
	if err != nil {
		return 0, err
	}
	object, err := w.Objects.Get(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("get reference object: %w", err)
	}
	defer object.Close()
	tmp, err := os.CreateTemp("", "lyra-reference-*")
	if err != nil {
		return 0, err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := io.Copy(tmp, object); err != nil {
		tmp.Close()
		return 0, err
	}
	if err := tmp.Close(); err != nil {
		return 0, err
	}
	if _, err := w.Audio.Probe(ctx, name); err != nil {
		return 0, err
	}
	pcm, err := w.Audio.Normalize(ctx, name)
	if err != nil {
		return 0, err
	}
	samples, err := audio.PCM16LE(pcm)
	if err != nil {
		return 0, err
	}
	fps, err := fingerprint.Extract(samples)
	if err != nil {
		return 0, err
	}
	w.Log.Debug("fingerprints_generated", "track_id", trackID, "fingerprints", len(fps))
	if err := w.Store.StoreTrack(ctx, trackID, fingerprint.AlgorithmLandmarkV1, fps); err != nil {
		return 0, err
	}
	return len(fps), nil
}
