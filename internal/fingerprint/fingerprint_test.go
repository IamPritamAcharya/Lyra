package fingerprint

import (
	"errors"
	"math"
	"reflect"
	"testing"
)

func TestHashRoundTripBoundaries(t *testing.T) {
	for _, f1 := range []int{0, 255} {
		for df := -31; df <= 31; df++ {
			for _, dt := range []int{MinDTFrames, MaxDTFrames} {
				h, err := EncodeHash(f1, df, dt)
				if err != nil {
					t.Fatal(err)
				}
				gotF, gotDF, gotDT := DecodeHash(h)
				if gotF != f1 || gotDF != df || gotDT != dt {
					t.Fatalf("%d,%d,%d -> %d,%d,%d", f1, df, dt, gotF, gotDF, gotDT)
				}
			}
		}
	}
}

func TestHashRejectsInvalidValues(t *testing.T) {
	for _, tc := range [][3]int{{-1, 0, 2}, {256, 0, 2}, {1, -32, 2}, {1, 32, 2}, {1, 0, 1}, {1, 0, 64}} {
		if _, err := EncodeHash(tc[0], tc[1], tc[2]); err == nil {
			t.Fatalf("accepted %#v", tc)
		}
	}
}

func TestExtractDeterministic(t *testing.T) {
	samples := tone(8, 440, 880, 1330)
	a, err := Extract(samples)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Extract(samples)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) == 0 {
		t.Fatal("expected fingerprints")
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatal("non-deterministic fingerprints")
	}
}

func TestExtractSilenceIsInsufficient(t *testing.T) {
	_, err := Extract(make([]int16, SampleRate*3))
	if !errors.Is(err, ErrInsufficientSignal) {
		t.Fatalf("got %v", err)
	}
}

func tone(seconds int, freqs ...float64) []int16 {
	s := make([]int16, seconds*SampleRate)
	for i := range s {
		v := 0.0
		for j, f := range freqs {
			v += 0.25 * math.Sin(2*math.Pi*f*float64(i)/SampleRate+float64(j)*0.3)
		}
		s[i] = int16(v * 28000)
	}
	return s
}
