package eval

import (
	"context"
	"testing"
)

type fake struct{ byPath map[string]Result }

func (f fake) Identify(_ context.Context, path string) (Result, error) { return f.byPath[path], nil }
func TestRunMeasuresCorrectAndNegative(t *testing.T) {
	r, err := Run(context.Background(), Manifest{Queries: []Query{{Path: "a", ExpectedMatch: true, ExpectedTrackID: "one"}, {Path: "b", ExpectedMatch: false}}}, fake{map[string]Result{"a": {Matched: true, TrackID: "one"}, "b": {Matched: false}}})
	if err != nil || r.Correct != 1 || r.CorrectNoMatch != 1 || r.FalsePositives != 0 {
		t.Fatalf("%+v %v", r, err)
	}
}
