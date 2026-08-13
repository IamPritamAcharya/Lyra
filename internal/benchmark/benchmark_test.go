package benchmark

import (
	"context"
	"testing"
)

func TestRun(t *testing.T) {
	r, err := Run(context.Background(), 100)
	if err != nil || !r.Matched || r.SyntheticTracks != 100 {
		t.Fatalf("%+v %v", r, err)
	}
}
