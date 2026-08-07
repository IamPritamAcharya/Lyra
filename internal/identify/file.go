package identify

import (
	"context"
	"fmt"
	"github.com/lyra/lyra/internal/audio"
	"github.com/lyra/lyra/internal/fingerprint"
)

// FileIdentifier composes safe audio preprocessing with the pure matcher.
type FileIdentifier struct {
	Audio   audio.Processor
	Matcher *Service
}

func (f FileIdentifier) IdentifyFile(ctx context.Context, path string) (Result, error) {
	if _, err := f.Audio.Probe(ctx, path); err != nil {
		return Result{}, err
	}
	pcm, err := f.Audio.Normalize(ctx, path)
	if err != nil {
		return Result{}, err
	}
	samples, err := audio.PCM16LE(pcm)
	if err != nil {
		return Result{}, err
	}
	fps, err := fingerprint.Extract(samples)
	if err != nil {
		return Result{}, err
	}
	result, err := f.Matcher.Match(ctx, fps)
	if err != nil && err != ErrNoMatch {
		return Result{}, fmt.Errorf("match query: %w", err)
	}
	return result, err
}
