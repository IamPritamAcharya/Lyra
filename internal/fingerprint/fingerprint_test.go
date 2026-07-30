package fingerprint

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
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

func TestLandmarkV1GoldenVector(t *testing.T) {
	fps, err := Extract(tone(3, 440, 880, 1330))
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, len(fps)*8)
	for i, fp := range fps {
		binary.LittleEndian.PutUint32(buf[i*8:], fp.Hash)
		binary.LittleEndian.PutUint32(buf[i*8+4:], uint32(fp.AnchorFrame))
	}
	got := sha256.Sum256(buf)
	const want = "8bb753c351bf6ecd7e81ba0b83e3d8bfc3a3c470ec935a9450ab05edeb98c2bf"
	if fmt.Sprintf("%x", got) != want || len(fps) != 244 {
		t.Fatalf("golden mismatch: sha256=%x count=%d", got, len(fps))
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
