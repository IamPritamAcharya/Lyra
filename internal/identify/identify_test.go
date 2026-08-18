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

type filteringIndex struct {
	fakeIndex
	allowed []uint32
}

func (f filteringIndex) FilterHashes(_ context.Context, _ int16, _ []uint32, _ int64) ([]uint32, error) {
	return f.allowed, nil
}
func TestTemporalAlignmentWinsOverScatteredHits(t *testing.T) {
	query := make([]fingerprint.Fingerprint, 10)
	posts := make([]Posting, 0, 20)
	for i := range query {
		hash, err := fingerprint.EncodeHash(i+1, 0, 2)
		if err != nil {
			t.Fatal(err)
		}
		query[i] = fingerprint.Fingerprint{Hash: hash, AnchorFrame: i * 3}
		posts = append(posts, Posting{Hash: hash, TrackID: 1, AnchorFrame: 100 + i*3})
		posts = append(posts, Posting{Hash: hash, TrackID: 2, AnchorFrame: 200 + i*7})
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

func TestCandidateEvidenceUsesOnlyWinningOffset(t *testing.T) {
	query := []fingerprint.Fingerprint{}
	postings := []Posting{}
	for i, frame := range []int{0, 10, 20, 30} {
		hash, err := fingerprint.EncodeHash(i+1, 0, 2)
		if err != nil {
			t.Fatal(err)
		}
		query = append(query, fingerprint.Fingerprint{Hash: hash, AnchorFrame: frame})
		postings = append(postings, Posting{Hash: hash, TrackID: 1, AnchorFrame: 100 + frame})
	}
	// This extra hit has a different offset and must not inflate the winning evidence.
	postings = append(postings, Posting{Hash: query[3].Hash, TrackID: 1, AnchorFrame: 400})
	cfg := DefaultConfig()
	cfg.MinAlignedHits = 1
	cfg.MinDistinctHashes = 1
	cfg.MinAnchors = 1
	cfg.MinDistinctFrequencyBins = 1
	cfg.MinAlignmentSpanFrames = 0
	r, err := New(&fakeIndex{postings: postings}, cfg).Match(context.Background(), query)
	if err != nil || !r.Matched {
		t.Fatalf("result=%+v err=%v", r, err)
	}
	got := r.Candidate
	if got.RawHashHits != 5 || got.AlignedHits != 4 || got.UniqueAlignedHashes != 4 || got.UniqueQueryAnchors != 4 || got.AlignmentSpanFrames != 30 {
		t.Fatalf("candidate=%+v", got)
	}
	if got.QueryAnchorCoverage != 1 || got.DistinctFrequencyBins != 4 {
		t.Fatalf("candidate=%+v", got)
	}
}

func TestAmbiguousTopCandidatesAreRejected(t *testing.T) {
	query := make([]fingerprint.Fingerprint, 6)
	postings := make([]Posting, 0, 12)
	for i := range query {
		hash, err := fingerprint.EncodeHash(i+1, 0, 2)
		if err != nil {
			t.Fatal(err)
		}
		query[i] = fingerprint.Fingerprint{Hash: hash, AnchorFrame: i * 5}
		for _, trackID := range []int64{1, 2} {
			postings = append(postings, Posting{Hash: hash, TrackID: trackID, AnchorFrame: int(trackID)*100 + i*5})
		}
	}
	r, err := New(&fakeIndex{postings: postings}, DefaultConfig()).Match(context.Background(), query)
	if err != ErrNoMatch || r.Matched {
		t.Fatalf("result=%+v err=%v", r, err)
	}
}

func TestSingleFrequencyEvidenceIsRejected(t *testing.T) {
	hash, err := fingerprint.EncodeHash(40, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	query := make([]fingerprint.Fingerprint, 6)
	postings := make([]Posting, 0, len(query))
	for i := range query {
		query[i] = fingerprint.Fingerprint{Hash: hash, AnchorFrame: i * 5}
		postings = append(postings, Posting{Hash: hash, TrackID: 1, AnchorFrame: 100 + i*5})
	}
	r, err := New(&fakeIndex{postings: postings}, DefaultConfig()).Match(context.Background(), query)
	if err != ErrNoMatch || r.Matched {
		t.Fatalf("result=%+v err=%v", r, err)
	}
}

func TestWeakCoverageAndAmbiguousLeadAreRejected(t *testing.T) {
	query := make([]fingerprint.Fingerprint, 100)
	posts := make([]Posting, 0, 24)
	for i := range query {
		hash, err := fingerprint.EncodeHash(i+1, 0, 2)
		if err != nil {
			t.Fatal(err)
		}
		query[i] = fingerprint.Fingerprint{Hash: hash, AnchorFrame: i * 3}
		if i < 6 {
			posts = append(posts, Posting{Hash: hash, TrackID: 1, AnchorFrame: 100 + i*3})
		}
	}
	cfg := DefaultConfig()
	cfg.MinQueryAnchorCoverage = 0.1
	r, err := New(&fakeIndex{postings: posts}, cfg).Match(context.Background(), query)
	if err != ErrNoMatch || r.Matched || len(r.Candidates) != 1 {
		t.Fatalf("weak coverage result=%+v err=%v", r, err)
	}

	posts = posts[:0]
	for i := 0; i < 12; i++ {
		for _, trackID := range []int64{1, 2} {
			posts = append(posts, Posting{Hash: query[i].Hash, TrackID: trackID, AnchorFrame: int(trackID)*100 + i*3})
		}
	}
	cfg = DefaultConfig()
	cfg.MinQueryAnchorCoverage = 0
	r, err = New(&fakeIndex{postings: posts}, cfg).Match(context.Background(), query)
	if err != ErrNoMatch || r.Matched || len(r.Candidates) != 2 {
		t.Fatalf("ambiguous lead result=%+v err=%v", r, err)
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
	cfg.MinDistinctFrequencyBins = 1
	cfg.MinAlignmentSpanFrames = 0
	cfg.MinCandidateLead = 0
	cfg.MaxCandidates = 1
	r, err := New(idx, cfg).Match(context.Background(), query)
	if err != nil || len(r.Candidates) != 1 {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestFingerprintLimitSamplesAcrossTheQuery(t *testing.T) {
	query := make([]fingerprint.Fingerprint, 20)
	posts := make([]Posting, 0, len(query))
	for i := range query {
		hash, err := fingerprint.EncodeHash(i+1, 0, 2)
		if err != nil {
			t.Fatal(err)
		}
		query[i] = fingerprint.Fingerprint{Hash: hash, AnchorFrame: i * 3}
		posts = append(posts, Posting{Hash: hash, TrackID: 1, AnchorFrame: 100 + i*3})
	}
	cfg := DefaultConfig()
	cfg.MaxFingerprints = 10
	r, err := New(&fakeIndex{postings: posts}, cfg).Match(context.Background(), query)
	if err != nil || !r.Matched || r.Candidate.TrackID != 1 || r.Candidate.UniqueQueryAnchors != 10 {
		t.Fatalf("result=%+v err=%v", r, err)
	}
}

func TestCommonHashFilterSuppressesWeakHashes(t *testing.T) {
	query := []fingerprint.Fingerprint{{Hash: 1, AnchorFrame: 1}}
	cfg := DefaultConfig()
	cfg.MinAlignedHits = 1
	cfg.MinDistinctHashes = 1
	cfg.MinAnchors = 1
	_, err := New(&filteringIndex{fakeIndex: fakeIndex{postings: []Posting{{Hash: 1, TrackID: 1, AnchorFrame: 2}}}}, cfg).Match(context.Background(), query)
	if err != ErrNoMatch {
		t.Fatalf("err=%v", err)
	}
}
