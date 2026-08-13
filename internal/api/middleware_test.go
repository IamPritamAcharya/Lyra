package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecureHeaders(t *testing.T) {
	r := httptest.NewRecorder()
	secureHeaders(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/", nil))
	if r.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff")
	}
}
func TestPanicRecovery(t *testing.T) {
	r := httptest.NewRecorder()
	recoverPanic(slog.New(slog.NewTextHandler(io.Discard, nil)), http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("test") })).ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/", nil))
	if r.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", r.Code)
	}
}
