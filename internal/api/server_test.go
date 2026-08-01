package api

import (
	"bytes"
	"encoding/json"
	"github.com/lyra/lyra/internal/catalog"
	"github.com/lyra/lyra/internal/config"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminCatalogAuthAndCreate(t *testing.T) {
	h := New(config.Config{Security: config.SecurityConfig{AdminAPIKey: "secret"}}, slog.New(slog.NewTextHandler(io.Discard, nil)), catalog.NewMemoryRepository(), func(*http.Request) error { return nil })
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/tracks", bytes.NewBufferString(`{"title":"A","artist":"B"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauth=%d", w.Code)
	}
	r.Header.Set("X-Lyra-Admin-Key", "secret")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAdminDelete(t *testing.T) {
	h := New(config.Config{Security: config.SecurityConfig{AdminAPIKey: "secret"}}, slog.New(slog.NewTextHandler(io.Discard, nil)), catalog.NewMemoryRepository(), func(*http.Request) error { return nil })
	create := httptest.NewRequest(http.MethodPost, "/v1/admin/tracks", bytes.NewBufferString(`{"title":"A","artist":"B"}`))
	create.Header.Set("X-Lyra-Admin-Key", "secret")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, create)
	var created struct{ PublicID string }
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodDelete, "/v1/admin/tracks/"+created.PublicID, nil)
	request.Header.Set("X-Lyra-Admin-Key", "secret")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, request)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete=%d body=%s", w.Code, w.Body.String())
	}
}
