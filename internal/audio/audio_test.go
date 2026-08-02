package audio

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRunner struct {
	out  []byte
	err  error
	args []string
}

func (f *fakeRunner) Output(_ context.Context, _ string, args ...string) ([]byte, error) {
	f.args = args
	return f.out, f.err
}
func TestProbeRejectsZeroDuration(t *testing.T) {
	f := &fakeRunner{out: []byte(`{"format":{"format_name":"wav","duration":"0"},"streams":[{"sample_rate":"11025","channels":1}]}`)}
	_, err := Processor{Runner: f, Timeout: time.Second, MaxDuration: time.Minute}.Probe(context.Background(), "x")
	if !errors.Is(err, ErrInvalidAudio) {
		t.Fatalf("%v", err)
	}
}
func TestNormalizeDoesNotUseShell(t *testing.T) {
	f := &fakeRunner{out: []byte{1, 0}}
	got, err := Processor{Runner: f, Timeout: time.Second}.Normalize(context.Background(), "name;unsafe")
	if err != nil || len(got) != 2 {
		t.Fatalf("%v %v", got, err)
	}
	if f.args[0] != "-nostdin" {
		t.Fatal(f.args)
	}
}
func TestPCM16LE(t *testing.T) {
	got, err := PCM16LE([]byte{0xff, 0xff, 1, 0})
	if err != nil || got[0] != -1 || got[1] != 1 {
		t.Fatalf("%v %v", got, err)
	}
}
