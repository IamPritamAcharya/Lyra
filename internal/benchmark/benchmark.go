// Package benchmark creates deterministic synthetic postings for matcher measurements.
package benchmark

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/lyra/lyra/internal/fingerprint"
	"github.com/lyra/lyra/internal/identify"
)

type Report struct {
	SyntheticTracks, QueryFingerprints, PostingCount int
	DurationMS                                       int64
	Matched                                          bool
}
type index struct{ postings []identify.Posting }

func (i index) Lookup(_ context.Context, _ int16, hashes []uint32) ([]identify.Posting, error) {
	wanted := map[uint32]struct{}{}
	for _, h := range hashes {
		wanted[h] = struct{}{}
	}
	out := make([]identify.Posting, 0)
	for _, p := range i.postings {
		if _, ok := wanted[p.Hash]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

// Run uses a fixed seed; it is a reproducible matcher benchmark, not a capacity claim.
func Run(ctx context.Context, tracks int) (Report, error) {
	if tracks <= 0 {
		return Report{}, fmt.Errorf("synthetic tracks must be positive")
	}
	rng := rand.New(rand.NewSource(1))
	query := make([]fingerprint.Fingerprint, 200)
	for i := range query {
		query[i] = fingerprint.Fingerprint{Hash: uint32(i + 1), AnchorFrame: i * 3}
	}
	postings := make([]identify.Posting, 0, tracks*4+len(query))
	for track := 1; track <= tracks; track++ {
		for n := 0; n < 4; n++ {
			postings = append(postings, identify.Posting{Hash: uint32(rng.Intn(len(query)) + 1), TrackID: int64(track), AnchorFrame: rng.Intn(4000)})
		}
	}
	for _, q := range query {
		postings = append(postings, identify.Posting{Hash: q.Hash, TrackID: 1, AnchorFrame: 1000 + q.AnchorFrame})
	}
	started := time.Now()
	result, err := identify.New(index{postings}, identify.DefaultConfig()).Match(ctx, query)
	return Report{SyntheticTracks: tracks, QueryFingerprints: len(query), PostingCount: len(postings), DurationMS: time.Since(started).Milliseconds(), Matched: result.Matched}, err
}
