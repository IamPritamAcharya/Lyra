// Package eval runs reproducible identification manifests and reports measured metrics.
package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

type Manifest struct {
	Queries []Query `json:"queries"`
}
type Query struct {
	Path            string `json:"path"`
	ExpectedTrackID string `json:"expected_track_id"`
	Condition       string `json:"condition"`
	ExpectedMatch   bool   `json:"expected_match"`
	DurationSeconds int    `json:"duration_seconds"`
}
type Result struct {
	Matched bool
	TrackID string
}
type Identifier interface {
	Identify(context.Context, string) (Result, error)
}
type Report struct {
	QueryCount, ExpectedMatches, Correct, WrongMatches, FalseNegatives, FalsePositives, ExpectedNoMatch, CorrectNoMatch int
	P50MS, P95MS, P99MS                                                                                                 int64
	Conditions                                                                                                          []ConditionReport
}
type ConditionReport struct {
	Condition                                                                                                           string
	QueryCount, ExpectedMatches, Correct, WrongMatches, FalseNegatives, FalsePositives, ExpectedNoMatch, CorrectNoMatch int
}

func Load(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return m, err
	}
	if len(m.Queries) == 0 {
		return m, fmt.Errorf("manifest has no queries")
	}
	return m, nil
}
func Run(ctx context.Context, m Manifest, id Identifier) (Report, error) {
	r := Report{QueryCount: len(m.Queries)}
	lat := make([]int64, 0, len(m.Queries))
	conditions := map[string]*ConditionReport{}
	for _, q := range m.Queries {
		start := time.Now()
		got, err := id.Identify(ctx, q.Path)
		lat = append(lat, time.Since(start).Milliseconds())
		if err != nil {
			return r, fmt.Errorf("identify %s: %w", q.Path, err)
		}
		condition := q.Condition
		if condition == "" {
			condition = "unspecified"
		}
		byCondition := conditions[condition]
		if byCondition == nil {
			byCondition = &ConditionReport{Condition: condition}
			conditions[condition] = byCondition
		}
		classify(&r, q, got)
		classifyCondition(byCondition, q, got)
	}
	sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
	if len(lat) > 0 {
		r.P50MS = lat[(len(lat)-1)*50/100]
		r.P95MS = lat[(len(lat)-1)*95/100]
		r.P99MS = lat[(len(lat)-1)*99/100]
	}
	r.Conditions = make([]ConditionReport, 0, len(conditions))
	for _, condition := range conditions {
		r.Conditions = append(r.Conditions, *condition)
	}
	sort.Slice(r.Conditions, func(i, j int) bool { return r.Conditions[i].Condition < r.Conditions[j].Condition })
	return r, nil
}

func classify(r *Report, q Query, got Result) {
	if !q.ExpectedMatch {
		r.ExpectedNoMatch++
		if !got.Matched {
			r.CorrectNoMatch++
		} else {
			r.FalsePositives++
		}
		return
	}
	r.ExpectedMatches++
	if !got.Matched {
		r.FalseNegatives++
		return
	}
	if got.TrackID == q.ExpectedTrackID {
		r.Correct++
		return
	}
	r.WrongMatches++
}

func classifyCondition(r *ConditionReport, q Query, got Result) {
	r.QueryCount++
	if !q.ExpectedMatch {
		r.ExpectedNoMatch++
		if !got.Matched {
			r.CorrectNoMatch++
		} else {
			r.FalsePositives++
		}
		return
	}
	r.ExpectedMatches++
	if !got.Matched {
		r.FalseNegatives++
		return
	}
	if got.TrackID == q.ExpectedTrackID {
		r.Correct++
		return
	}
	r.WrongMatches++
}
