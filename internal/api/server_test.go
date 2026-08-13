package api

import (
	"bytes"
	"context"
	"encoding/json"
	authdomain "github.com/lyra/lyra/internal/auth"
	"github.com/lyra/lyra/internal/catalog"
	"github.com/lyra/lyra/internal/config"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type testAdminAuth struct{}

func (testAdminAuth) Login(_ context.Context, username, password string) (string, string, time.Time, error) {
	if username != "admin" || password != "password" {
		return "", "", time.Time{}, authdomain.ErrInvalidCredentials
	}
	return "session-token", "csrf-token", time.Now().Add(time.Hour), nil
}
func (testAdminAuth) Validate(_ context.Context, token string) error {
	if token != "session-token" {
		return authdomain.ErrUnauthenticated
	}
	return nil
}
func (testAdminAuth) ValidateCSRF(_ context.Context, token, csrf string) error {
	if token != "session-token" || csrf != "csrf-token" {
		return authdomain.ErrCSRF
	}
	return nil
}
func (testAdminAuth) Logout(context.Context, string) error { return nil }

func TestAdminCatalogAuthAndCreate(t *testing.T) {
	h := New(config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), catalog.NewMemoryRepository(), func(*http.Request) error { return nil }, nil, nil, testAdminAuth{})
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/tracks", bytes.NewBufferString(`{"title":"A","artist":"B"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauth=%d", w.Code)
	}
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-token"})
	r.Header.Set("X-CSRF-Token", "csrf-token")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAdminDelete(t *testing.T) {
	h := New(config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), catalog.NewMemoryRepository(), func(*http.Request) error { return nil }, nil, nil, testAdminAuth{})
	create := httptest.NewRequest(http.MethodPost, "/v1/admin/tracks", bytes.NewBufferString(`{"title":"A","artist":"B"}`))
	create.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-token"})
	create.Header.Set("X-CSRF-Token", "csrf-token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, create)
	var created struct{ PublicID string }
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodDelete, "/v1/admin/tracks/"+created.PublicID, nil)
	request.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-token"})
	request.Header.Set("X-CSRF-Token", "csrf-token")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, request)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAdminLoginSetsHttpOnlyCookie(t *testing.T) {
	h := New(config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), catalog.NewMemoryRepository(), func(*http.Request) error { return nil }, nil, nil, testAdminAuth{})
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/auth/login", bytes.NewBufferString(`{"username":"admin","password":"password"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("login=%d body=%s", w.Code, w.Body.String())
	}
	if got := w.Result().Cookies(); len(got) != 1 || !got[0].HttpOnly || got[0].Name != sessionCookie {
		t.Fatalf("unexpected cookie: %#v", got)
	}
}
