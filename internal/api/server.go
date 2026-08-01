package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/lyra/lyra/internal/catalog"
	"github.com/lyra/lyra/internal/config"
)

func New(cfg config.Config, log *slog.Logger, repo catalog.Repository, ready func(*http.Request) error) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", jsonHandler(func(http.ResponseWriter, *http.Request) (any, error) { return map[string]string{"status": "live"}, nil }))
	mux.HandleFunc("GET /health/ready", jsonHandler(func(_ http.ResponseWriter, r *http.Request) (any, error) {
		if err := ready(r); err != nil {
			return nil, err
		}
		return map[string]string{"status": "ready"}, nil
	}))
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte("# HELP lyra_http_requests_total Requests handled by Lyra\n# TYPE lyra_http_requests_total counter\nlyra_http_requests_total 0\n"))
	})
	admin := http.NewServeMux()
	admin.HandleFunc("POST /v1/admin/tracks", createTrack(repo))
	admin.HandleFunc("GET /v1/admin/tracks", listTracks(repo))
	admin.HandleFunc("GET /v1/admin/tracks/{id}", getTrack(repo))
	admin.HandleFunc("DELETE /v1/admin/tracks/{id}", deleteTrack(repo))
	mux.Handle("/v1/admin/", requireAdmin(cfg.Security.AdminAPIKey, admin))
	return requestLog(log, mux)
}

func createTrack(repo catalog.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Title  string  `json:"title"`
			Artist string  `json:"artist"`
			Album  *string `json:"album"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		t, err := repo.Create(r.Context(), catalog.CreateTrack{Title: in.Title, ArtistName: in.Artist, AlbumName: in.Album})
		writeResult(w, t, err, http.StatusCreated)
	}
}
func listTracks(repo catalog.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > 100 {
			limit = 50
		}
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		if offset < 0 {
			offset = 0
		}
		v, err := repo.List(r.Context(), limit, offset)
		writeResult(w, v, err, http.StatusOK)
	}
}
func getTrack(repo catalog.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, err := repo.Get(r.Context(), r.PathValue("id"))
		writeResult(w, v, err, http.StatusOK)
	}
}
func deleteTrack(repo catalog.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		track, err := repo.Get(r.Context(), id)
		if err != nil {
			writeResult(w, nil, err, http.StatusNoContent)
			return
		}
		if track.Status != catalog.Deleting && track.Status != catalog.Deleted {
			track, err = repo.Transition(r.Context(), id, catalog.Deleting, nil)
		}
		if err == nil {
			track, err = repo.Transition(r.Context(), id, catalog.Deleted, nil)
		}
		writeResult(w, track, err, http.StatusNoContent)
	}
}
func requireAdmin(key string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Lyra-Admin-Key")
		if key == "" || subtle.ConstantTimeCompare([]byte(got), []byte(key)) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func jsonHandler(fn func(http.ResponseWriter, *http.Request) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { v, e := fn(w, r); writeResult(w, v, e, http.StatusOK) }
}
func writeResult(w http.ResponseWriter, v any, err error, status int) {
	if err != nil {
		if errors.Is(err, catalog.ErrTrackNotFound) {
			writeError(w, http.StatusNotFound, "track_not_found")
		} else if status == http.StatusOK {
			writeError(w, http.StatusServiceUnavailable, "not_ready")
		} else {
			writeError(w, http.StatusBadRequest, "invalid_request")
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}
func requestLog(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Info("http_request_completed", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(started).Milliseconds())
	})
}
