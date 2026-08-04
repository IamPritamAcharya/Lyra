package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

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
}

func (w Worker) HandleFingerprintTask(ctx context.Context, task *asynq.Task) error {
	var payload queue.FingerprintTrackPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("decode fingerprint task: %w", err)
	}
	err := w.index(ctx, payload.TrackID)
	if err != nil && w.Failures != nil {
		_ = w.Failures.Fail(ctx, payload.TrackID, err.Error())
	}
	return err
}
func (w Worker) index(ctx context.Context, trackID string) error {
	key, err := w.Locate.ObjectKey(ctx, trackID)
	if err != nil {
		return err
	}
	object, err := w.Objects.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("get reference object: %w", err)
	}
	defer object.Close()
	tmp, err := os.CreateTemp("", "lyra-reference-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := io.Copy(tmp, object); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if _, err := w.Audio.Probe(ctx, name); err != nil {
		return err
	}
	pcm, err := w.Audio.Normalize(ctx, name)
	if err != nil {
		return err
	}
	samples, err := audio.PCM16LE(pcm)
	if err != nil {
		return err
	}
	fps, err := fingerprint.Extract(samples)
	if err != nil {
		return err
	}
	return w.Store.StoreTrack(ctx, trackID, fingerprint.AlgorithmLandmarkV1, fps)
}
