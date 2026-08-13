package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetricsExposePrometheusCounters(t *testing.T) {
	m := NewMetrics()
	m.ObserveHTTP(time.Millisecond)
	m.ObserveIdentify(2*time.Millisecond, true)
	r := httptest.NewRecorder()
	m.Handler().ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	for _, want := range []string{"lyra_http_requests_total 1", "lyra_identify_requests_total 1", "lyra_identify_matches_total 1"} {
		if !strings.Contains(r.Body.String(), want) {
			t.Fatalf("missing %q in %s", want, r.Body.String())
		}
	}
}
