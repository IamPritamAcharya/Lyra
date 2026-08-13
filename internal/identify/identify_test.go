package identify

import (
	"context"
	"github.com/lyra/lyra/internal/fingerprint"
	"testing"
)

type fakeIndex struct {
	postings []Posting
	calls    int
}

func (f *fakeIndex) Lookup(_ context.Context, _ int16, _ []uint32) ([]Posting, error) {
	f.calls++
	return f.postings, nil
}
func TestTemporalAlignmentWinsOverScatteredHits(t *testing.T) {
	query := make([]fingerprint.Fingerprint, 10)
	posts := make([]Posting, 0, 20)
	for i := range query {
		query[i] = fingerprint.Fingerprint{Hash: uint32(i + 1), AnchorFrame: i * 3}
		posts = append(posts, Posting{Hash: uint32(i + 1), TrackID: 1, AnchorFrame: 100 + i*3})
		posts = append(posts, Posting{Hash: uint32(i + 1), TrackID: 2, AnchorFrame: 200 + i*7})
	}
	idx := &fakeIndex{postings: posts}
	r, err := New(idx, DefaultConfig()).Match(context.Background(), query)
	if err != nil || !r.Matched || r.Candidate.TrackID != 1 {
		t.Fatalf("result=%+v err=%v", r, err)
	}
	if idx.calls != 1 {
		t.Fatalf("lookup calls=%d", idx.calls)
	}
}
func TestNoMatchAndInsufficient(t *testing.T) {
	_, err := New(&fakeIndex{}, DefaultConfig()).Match(context.Background(), nil)
	if err != fingerprint.ErrInsufficientSignal {
		t.Fatal(err)
	}
	r, err := New(&fakeIndex{}, DefaultConfig()).Match(context.Background(), []fingerprint.Fingerprint{{Hash: 1, AnchorFrame: 1}})
	if err != ErrNoMatch || r.Matched {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestSafeguardsBoundWork(t *testing.T) {
	query := []fingerprint.Fingerprint{{Hash: 1, AnchorFrame: 1}, {Hash: 2, AnchorFrame: 2}}
	idx := &fakeIndex{postings: []Posting{{Hash: 1, TrackID: 1, AnchorFrame: 10}, {Hash: 2, TrackID: 2, AnchorFrame: 10}}}
	cfg := DefaultConfig()
	cfg.MaxFingerprints = 1
	if _, err := New(idx, cfg).Match(context.Background(), query); err != ErrNoMatch {
		t.Fatalf("err=%v", err)
	}
	cfg = DefaultConfig()
	cfg.MinAlignedHits = 1
	cfg.MinDistinctHashes = 1
	cfg.MinAnchors = 1
	cfg.MaxCandidates = 1
	r, err := New(idx, cfg).Match(context.Background(), query)
	if err != nil || len(r.Candidates) != 1 {
		t.Fatalf("%+v %v", r, err)
	}
}
