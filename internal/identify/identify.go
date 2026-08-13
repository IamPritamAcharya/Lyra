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

// HashFilter is optional because statistics are a scalability optimization, not correctness state.
type HashFilter interface {
	FilterHashes(context.Context, int16, []uint32, int64) ([]uint32, error)
}
type Config struct {
	Version                                                                                                        int16
	AlignmentTolerance, MinAlignedHits, MinDistinctHashes, MinAnchors, MaxFingerprints, MaxPostings, MaxCandidates int
	MaxPostingsPerHash                                                                                             int64
}

func DefaultConfig() Config {
	return Config{Version: fingerprint.AlgorithmLandmarkV1, AlignmentTolerance: 2, MinAlignedHits: 6, MinDistinctHashes: 3, MinAnchors: 3, MaxFingerprints: 5000, MaxPostings: 50000, MaxCandidates: 1000, MaxPostingsPerHash: 10000}
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
	if s.cfg.MaxFingerprints > 0 && len(query) > s.cfg.MaxFingerprints {
		return Result{}, ErrNoMatch
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
	sort.Slice(hashes, func(i, j int) bool { return hashes[i] < hashes[j] })
	if filter, ok := s.index.(HashFilter); ok && s.cfg.MaxPostingsPerHash > 0 {
		var err error
		hashes, err = filter.FilterHashes(ctx, s.cfg.Version, hashes, s.cfg.MaxPostingsPerHash)
		if err != nil {
			return Result{}, err
		}
		if len(hashes) == 0 {
			return Result{}, ErrNoMatch
		}
	}
	postings, err := s.index.Lookup(ctx, s.cfg.Version, hashes)
	if err != nil {
		return Result{}, err
	}
	states := map[int64]*state{}
	postingsSeen := 0
	for _, p := range postings {
		postingsSeen++
		if s.cfg.MaxPostings > 0 && postingsSeen > s.cfg.MaxPostings {
			break
		}
		for frame := range anchors[p.Hash] {
			st := states[p.TrackID]
			if st == nil {
				if s.cfg.MaxCandidates > 0 && len(states) >= s.cfg.MaxCandidates {
					continue
				}
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
		if a.AlignmentCoherence != b.AlignmentCoherence {
			return a.AlignmentCoherence > b.AlignmentCoherence
		}
		return a.TrackID < b.TrackID
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
