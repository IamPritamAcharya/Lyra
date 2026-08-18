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
	Version                                                                                                                               int16
	AlignmentTolerance, MinAlignedHits, MinDistinctHashes, MinAnchors, MinDistinctFrequencyBins, MinAlignmentSpanFrames, MinCandidateLead int
	MinQueryAnchorCoverage                                                                                                                float64
	MaxFingerprints, MaxPostings, MaxCandidates                                                                                           int
	MaxPostingsPerHash                                                                                                                    int64
}

func DefaultConfig() Config {
	return Config{
		Version:                  fingerprint.AlgorithmLandmarkV1,
		AlignmentTolerance:       2,
		MinAlignedHits:           6,
		MinDistinctHashes:        3,
		MinAnchors:               3,
		MinDistinctFrequencyBins: 2,
		MinAlignmentSpanFrames:   4,
		MinQueryAnchorCoverage:   0.02,
		MinCandidateLead:         3,
		MaxFingerprints:          5000,
		MaxPostings:              50000,
		MaxCandidates:            1000,
		MaxPostingsPerHash:       10000,
	}
}

type Candidate struct {
	TrackID                                                                                  int64
	RawHashHits, AlignedHits, UniqueAlignedHashes, UniqueQueryAnchors, DistinctFrequencyBins int
	AlignmentSpanFrames, BestAlignmentOffset, RunnerUpAlignmentHits                          int
	AlignmentCoherence, QueryAnchorCoverage                                                  float64
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
	usableQueryAnchors := 0
	for _, hash := range hashes {
		usableQueryAnchors += len(anchors[hash])
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
				st = &state{offsets: map[int]int{}}
				states[p.TrackID] = st
			}
			offset := p.AnchorFrame - frame
			st.add(p.Hash, frame, offset)
		}
	}
	out := make([]Candidate, 0, len(states))
	for id, st := range states {
		best, bestOffset, runnerUp := st.bestAlignment(s.cfg.AlignmentTolerance)
		c := st.candidate(id, best, bestOffset, runnerUp, s.cfg.AlignmentTolerance, usableQueryAnchors)
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
		if a.DistinctFrequencyBins != b.DistinctFrequencyBins {
			return a.DistinctFrequencyBins > b.DistinctFrequencyBins
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
	if best.AlignedHits < s.cfg.MinAlignedHits ||
		best.UniqueAlignedHashes < s.cfg.MinDistinctHashes ||
		best.UniqueQueryAnchors < s.cfg.MinAnchors ||
		best.DistinctFrequencyBins < s.cfg.MinDistinctFrequencyBins ||
		best.AlignmentSpanFrames < s.cfg.MinAlignmentSpanFrames ||
		best.QueryAnchorCoverage < s.cfg.MinQueryAnchorCoverage {
		return result, ErrNoMatch
	}
	if len(out) > 1 && best.AlignedHits-out[1].AlignedHits < s.cfg.MinCandidateLead {
		return result, ErrNoMatch
	}
	result.Matched = true
	result.Candidate = &best
	return result, nil
}

type state struct {
	raw     int
	offsets map[int]int
	matches []match
}

type match struct {
	hash, queryAnchor, offset int
}

func (s *state) add(hash uint32, queryAnchor, offset int) {
	s.raw++
	s.offsets[offset]++
	s.matches = append(s.matches, match{hash: int(hash), queryAnchor: queryAnchor, offset: offset})
}

func (s state) bestAlignment(tolerance int) (best, bestOffset, runnerUp int) {
	for offset := range s.offsets {
		score := s.alignmentScore(offset, tolerance)
		if score > best || (score == best && offset < bestOffset) {
			best, bestOffset = score, offset
		}
	}
	for offset := range s.offsets {
		if abs(offset-bestOffset) <= 2*tolerance {
			continue
		}
		if score := s.alignmentScore(offset, tolerance); score > runnerUp {
			runnerUp = score
		}
	}
	return best, bestOffset, runnerUp
}

func (s state) alignmentScore(offset, tolerance int) int {
	score := 0
	for delta := -tolerance; delta <= tolerance; delta++ {
		score += s.offsets[offset+delta]
	}
	return score
}

func (s state) candidate(trackID int64, alignedHits, bestOffset, runnerUp, tolerance, usableQueryAnchors int) Candidate {
	hashes := map[int]struct{}{}
	anchors := map[int]struct{}{}
	frequencies := map[int]struct{}{}
	spanMin, spanMax := 0, 0
	first := true
	for _, evidence := range s.matches {
		if abs(evidence.offset-bestOffset) > tolerance {
			continue
		}
		hashes[evidence.hash] = struct{}{}
		anchors[evidence.queryAnchor] = struct{}{}
		frequency, _, _ := fingerprint.DecodeHash(uint32(evidence.hash))
		frequencies[frequency] = struct{}{}
		if first || evidence.queryAnchor < spanMin {
			spanMin = evidence.queryAnchor
		}
		if first || evidence.queryAnchor > spanMax {
			spanMax = evidence.queryAnchor
		}
		first = false
	}
	candidate := Candidate{
		TrackID:               trackID,
		RawHashHits:           s.raw,
		AlignedHits:           alignedHits,
		UniqueAlignedHashes:   len(hashes),
		UniqueQueryAnchors:    len(anchors),
		DistinctFrequencyBins: len(frequencies),
		AlignmentSpanFrames:   spanMax - spanMin,
		BestAlignmentOffset:   bestOffset,
		RunnerUpAlignmentHits: runnerUp,
	}
	if usableQueryAnchors > 0 {
		candidate.QueryAnchorCoverage = float64(candidate.UniqueQueryAnchors) / float64(usableQueryAnchors)
	}
	return candidate
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
