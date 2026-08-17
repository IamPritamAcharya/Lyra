package eval

import (
	"context"
	"testing"
)

type fake struct{ byPath map[string]Result }

func (f fake) Identify(_ context.Context, path string) (Result, error) { return f.byPath[path], nil }
func TestRunMeasuresCorrectAndNegative(t *testing.T) {
	r, err := Run(context.Background(), Manifest{Queries: []Query{{Path: "a", ExpectedMatch: true, ExpectedTrackID: "one"}, {Path: "b", ExpectedMatch: false}}}, fake{map[string]Result{"a": {Matched: true, TrackID: "one"}, "b": {Matched: false}}})
	if err != nil || r.ExpectedMatches != 1 || r.Correct != 1 || r.CorrectNoMatch != 1 || r.FalsePositives != 0 || r.FalseNegatives != 0 {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestRunReportsWrongMatchesAndConditions(t *testing.T) {
	r, err := Run(context.Background(), Manifest{Queries: []Query{
		{Path: "clean", ExpectedMatch: true, ExpectedTrackID: "one", Condition: "clean"},
		{Path: "noise", ExpectedMatch: true, ExpectedTrackID: "one", Condition: "noise_10db"},
		{Path: "negative", ExpectedMatch: false, Condition: "no_match"},
	}}, fake{map[string]Result{
		"clean":    {Matched: true, TrackID: "one"},
		"noise":    {Matched: true, TrackID: "two"},
		"negative": {Matched: true, TrackID: "three"},
	}})
	if err != nil || r.QueryCount != 3 || r.ExpectedMatches != 2 || r.Correct != 1 || r.WrongMatches != 1 || r.FalsePositives != 1 || r.FalseNegatives != 0 || len(r.Conditions) != 3 {
		t.Fatalf("%+v %v", r, err)
	}
	if r.Conditions[0].Condition != "clean" || r.Conditions[1].Condition != "no_match" || r.Conditions[2].Condition != "noise_10db" {
		t.Fatalf("conditions=%+v", r.Conditions)
	}
}
