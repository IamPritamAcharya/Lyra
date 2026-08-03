// Package identify performs deterministic retrieval and temporal alignment.
package identify

import (
	"context"
	"errors"
	"sort"

	"github.com/lyra/lyra/internal/fingerprint"
)

var ErrNoMatch = errors.New("no match")

type Posting struct {
	Hash        uint32
	TrackID     int64
	AnchorFrame int
}

// FingerprintIndex is deliberately owned by the matcher; Postgres implements it first.
type FingerprintIndex interface {
	Lookup(context.Context, int16, []uint32) ([]Posting, error)
}
type Config struct {
	Version                                                           int16
	AlignmentTolerance, MinAlignedHits, MinDistinctHashes, MinAnchors int
}

func DefaultConfig() Config {
	return Config{Version: fingerprint.AlgorithmLandmarkV1, AlignmentTolerance: 2, MinAlignedHits: 6, MinDistinctHashes: 3, MinAnchors: 3}
}

type Candidate struct {
	TrackID                                                           int64
	RawHashHits, AlignedHits, UniqueAlignedHashes, UniqueQueryAnchors int
	AlignmentSpanFrames, BestAlignmentOffset                          int
	AlignmentCoherence                                                float64
}
type Result struct {
	Matched    bool
	Candidate  *Candidate
	Candidates []Candidate
}
type Service struct {
	index FingerprintIndex
	cfg   Config
}

func New(index FingerprintIndex, cfg Config) *Service { return &Service{index: index, cfg: cfg} }

func (s *Service) Match(ctx context.Context, query []fingerprint.Fingerprint) (Result, error) {
	if len(query) == 0 {
		return Result{}, fingerprint.ErrInsufficientSignal
	}
	anchors := map[uint32]map[int]struct{}{}
	for _, fp := range query {
		if anchors[fp.Hash] == nil {
			anchors[fp.Hash] = map[int]struct{}{}
		}
		anchors[fp.Hash][fp.AnchorFrame] = struct{}{}
	}
	hashes := make([]uint32, 0, len(anchors))
	for h := range anchors {
		hashes = append(hashes, h)
	}
	postings, err := s.index.Lookup(ctx, s.cfg.Version, hashes)
	if err != nil {
		return Result{}, err
	}
	states := map[int64]*state{}
	for _, p := range postings {
		for frame := range anchors[p.Hash] {
			st := states[p.TrackID]
			if st == nil {
				st = &state{offsets: map[int]int{}, hashes: map[uint32]struct{}{}, anchors: map[int]struct{}{}}
				states[p.TrackID] = st
			}
			st.raw++
			offset := p.AnchorFrame - frame
			st.offsets[offset]++
			st.hashes[p.Hash] = struct{}{}
			st.anchors[frame] = struct{}{}
		}
	}
	out := make([]Candidate, 0, len(states))
	for id, st := range states {
		best, bestOffset := 0, 0
		for offset := range st.offsets {
			score := 0
			for delta := -s.cfg.AlignmentTolerance; delta <= s.cfg.AlignmentTolerance; delta++ {
				score += st.offsets[offset+delta]
			}
			if score > best || (score == best && offset < bestOffset) {
				best, bestOffset = score, offset
			}
		}
		spanMin, spanMax := 0, 0
		first := true
		for frame := range st.anchors {
			if first || frame < spanMin {
				spanMin = frame
			}
			if first || frame > spanMax {
				spanMax = frame
			}
			first = false
		}
		c := Candidate{TrackID: id, RawHashHits: st.raw, AlignedHits: best, UniqueAlignedHashes: len(st.hashes), UniqueQueryAnchors: len(st.anchors), AlignmentSpanFrames: spanMax - spanMin, BestAlignmentOffset: bestOffset}
		if c.RawHashHits > 0 {
			c.AlignmentCoherence = float64(c.AlignedHits) / float64(c.RawHashHits)
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.AlignedHits != b.AlignedHits {
			return a.AlignedHits > b.AlignedHits
		}
		if a.UniqueAlignedHashes != b.UniqueAlignedHashes {
			return a.UniqueAlignedHashes > b.UniqueAlignedHashes
		}
		if a.AlignmentSpanFrames != b.AlignmentSpanFrames {
			return a.AlignmentSpanFrames > b.AlignmentSpanFrames
		}
		return a.AlignmentCoherence > b.AlignmentCoherence
	})
	result := Result{Candidates: out}
	if len(out) == 0 {
		return result, ErrNoMatch
	}
	best := out[0]
	if best.AlignedHits < s.cfg.MinAlignedHits || best.UniqueAlignedHashes < s.cfg.MinDistinctHashes || best.UniqueQueryAnchors < s.cfg.MinAnchors {
		return result, ErrNoMatch
	}
	result.Matched = true
	result.Candidate = &best
	return result, nil
}

type state struct {
	raw     int
	offsets map[int]int
	hashes  map[uint32]struct{}
	anchors map[int]struct{}
}
