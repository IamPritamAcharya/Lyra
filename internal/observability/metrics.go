// Package observability provides a small Prometheus-compatible metrics surface.
package observability

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	httpRequests, identifyRequests, identifyMatches, identifyNoMatches atomic.Uint64
	mu                                                                 sync.Mutex
	httpDurations, identifyDurations                                   []float64
}

func NewMetrics() *Metrics { return &Metrics{} }
func (m *Metrics) ObserveHTTP(d time.Duration) {
	m.httpRequests.Add(1)
	m.mu.Lock()
	m.httpDurations = append(m.httpDurations, d.Seconds())
	m.mu.Unlock()
}
func (m *Metrics) ObserveIdentify(d time.Duration, matched bool) {
	m.identifyRequests.Add(1)
	if matched {
		m.identifyMatches.Add(1)
	} else {
		m.identifyNoMatches.Add(1)
	}
	m.mu.Lock()
	m.identifyDurations = append(m.identifyDurations, d.Seconds())
	m.mu.Unlock()
}
func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		m.mu.Lock()
		httpD := append([]float64(nil), m.httpDurations...)
		identifyD := append([]float64(nil), m.identifyDurations...)
		m.mu.Unlock()
		fmt.Fprintf(w, "# TYPE lyra_http_requests_total counter\nlyra_http_requests_total %d\n", m.httpRequests.Load())
		fmt.Fprintf(w, "# TYPE lyra_identify_requests_total counter\nlyra_identify_requests_total %d\nlyra_identify_matches_total %d\nlyra_identify_no_matches_total %d\n", m.identifyRequests.Load(), m.identifyMatches.Load(), m.identifyNoMatches.Load())
		writeSummary(w, "lyra_http_request_duration_seconds", httpD)
		writeSummary(w, "lyra_identify_duration_seconds", identifyD)
	})
}
func writeSummary(w http.ResponseWriter, name string, values []float64) {
	fmt.Fprintf(w, "# TYPE %s summary\n", name)
	if len(values) == 0 {
		fmt.Fprintf(w, "%s_count 0\n%s_sum 0\n", name, name)
		return
	}
	sort.Float64s(values)
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	for _, q := range []float64{.5, .95, .99} {
		idx := int(float64(len(values)-1) * q)
		fmt.Fprintf(w, "%s{quantile=\"%g\"} %g\n", name, q, values[idx])
	}
	fmt.Fprintf(w, "%s_count %d\n%s_sum %g\n", name, len(values), name, sum)
}
func SanitizeLabel(s string) string { return strings.ReplaceAll(s, "\n", "") }
