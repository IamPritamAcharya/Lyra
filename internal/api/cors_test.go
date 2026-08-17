package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORSRestrictsConfiguredOrigin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/v1/identify", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	cors("http://localhost:5173", next).ServeHTTP(r, request)
	if r.Code != http.StatusNoContent || r.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("status=%d origin=%q", r.Code, r.Header().Get("Access-Control-Allow-Origin"))
	}
	if !strings.Contains(r.Header().Get("Access-Control-Allow-Headers"), "X-Lyra-Live-Capture-Ms") {
		t.Fatalf("live capture header is not permitted: %q", r.Header().Get("Access-Control-Allow-Headers"))
	}
}

func TestCORSAllowsEachExplicitConfiguredOrigin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	for _, origin := range []string{"http://localhost:5173", "https://phone.example"} {
		r := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodOptions, "/v1/identify", nil)
		request.Header.Set("Origin", origin)
		cors("http://localhost:5173, https://phone.example", next).ServeHTTP(r, request)
		if r.Header().Get("Access-Control-Allow-Origin") != origin {
			t.Fatalf("origin=%q allowed=%q", origin, r.Header().Get("Access-Control-Allow-Origin"))
		}
	}
}
