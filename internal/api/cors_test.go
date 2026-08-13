package api

import (
	"net/http"
	"net/http/httptest"
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
}
