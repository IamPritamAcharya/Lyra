// Package audio owns safe external audio inspection and canonical conversion.
package audio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

var (
	ErrInvalidAudio     = errors.New("invalid audio")
	ErrUnsupportedAudio = errors.New("unsupported audio")
)

type ProbeResult struct {
	FormatName           string
	Duration             time.Duration
	SampleRate, Channels int
}
type Runner interface {
	Output(context.Context, string, ...string) ([]byte, error)
}
type CommandRunner struct{}

func (CommandRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

type Processor struct {
	Runner      Runner
	Timeout     time.Duration
	MaxDuration time.Duration
}

func NewProcessor() Processor {
	return Processor{Runner: CommandRunner{}, Timeout: 15 * time.Second, MaxDuration: 20 * time.Minute}
}

func (p Processor) Probe(ctx context.Context, path string) (ProbeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()
	out, err := p.Runner.Output(ctx, "ffprobe", "-v", "error", "-show_entries", "format=format_name,duration:stream=sample_rate,channels", "-of", "json", path)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("ffprobe: %w", ErrInvalidAudio)
	}
	var raw struct {
		Format struct {
			FormatName string `json:"format_name"`
			Duration   string `json:"duration"`
		}
		Streams []struct {
			SampleRate string `json:"sample_rate"`
			Channels   int    `json:"channels"`
		}
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return ProbeResult{}, fmt.Errorf("decode ffprobe: %w", ErrInvalidAudio)
	}
	duration, err := strconv.ParseFloat(raw.Format.Duration, 64)
	if err != nil || duration <= 0 {
		return ProbeResult{}, fmt.Errorf("duration: %w", ErrInvalidAudio)
	}
	if time.Duration(duration*float64(time.Second)) > p.MaxDuration {
		return ProbeResult{}, fmt.Errorf("duration exceeds maximum: %w", ErrInvalidAudio)
	}
	if len(raw.Streams) == 0 {
		return ProbeResult{}, fmt.Errorf("no audio stream: %w", ErrUnsupportedAudio)
	}
	rate, _ := strconv.Atoi(raw.Streams[0].SampleRate)
	return ProbeResult{FormatName: raw.Format.FormatName, Duration: time.Duration(duration * float64(time.Second)), SampleRate: rate, Channels: raw.Streams[0].Channels}, nil
}

// Normalize returns canonical mono 11025Hz signed 16-bit PCM. It never uses a shell.
func (p Processor) Normalize(ctx context.Context, path string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()
	out, err := p.Runner.Output(ctx, "ffmpeg", "-nostdin", "-v", "error", "-i", path, "-ac", "1", "-ar", "11025", "-f", "s16le", "pipe:1")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg normalize: %w", ErrInvalidAudio)
	}
	if len(out) == 0 || len(out)%2 != 0 {
		return nil, fmt.Errorf("normalized pcm: %w", ErrInvalidAudio)
	}
	return out, nil
}
func PCM16LE(bytes []byte) ([]int16, error) {
	if len(bytes) == 0 || len(bytes)%2 != 0 {
		return nil, ErrInvalidAudio
	}
	out := make([]int16, len(bytes)/2)
	for i := range out {
		out[i] = int16(uint16(bytes[i*2]) | uint16(bytes[i*2+1])<<8)
	}
	return out, nil
}
