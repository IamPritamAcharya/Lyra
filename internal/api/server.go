package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/lyra/lyra/internal/catalog"
	"github.com/lyra/lyra/internal/config"
	"github.com/lyra/lyra/internal/observability"
)

func New(cfg config.Config, log *slog.Logger, repo catalog.Repository, ready func(*http.Request) error, identifier FileIdentifier, uploader ReferenceUploader, auth AdminAuth) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", jsonHandler(func(http.ResponseWriter, *http.Request) (any, error) { return map[string]string{"status": "live"}, nil }))
	mux.HandleFunc("GET /health/ready", jsonHandler(func(_ http.ResponseWriter, r *http.Request) (any, error) {
		if err := ready(r); err != nil {
			return nil, err
		}
		return map[string]string{"status": "ready"}, nil
	}))
	metrics := observability.NewMetrics()
	mux.Handle("GET /metrics", metrics.Handler())
	limit := cfg.Security.IdentifyPerMinute
	if limit <= 0 {
		limit = 30
	}
	mux.Handle("POST /v1/identify", newLimiter(limit).middleware(identifyHandler(cfg.Security.MaxIdentifyBytes, identifier, repo, metrics, log)))
	admin := http.NewServeMux()
	admin.HandleFunc("POST /v1/admin/tracks", createTrack(repo, log))
	admin.HandleFunc("GET /v1/admin/tracks", listTracks(repo))
	admin.HandleFunc("GET /v1/admin/tracks/{id}", getTrack(repo))
	admin.HandleFunc("DELETE /v1/admin/tracks/{id}", deleteTrack(repo, log))
	admin.HandleFunc("POST /v1/admin/tracks/{id}/audio", uploadTrackAudio(uploader, cfg.Security.MaxIdentifyBytes, log))
	if auth != nil {
		mux.Handle("POST /v1/admin/auth/login", newLimiter(5).middleware(login(auth, cfg.Security.AdminCookieSecure, log)))
		mux.HandleFunc("POST /v1/admin/auth/logout", logout(auth, cfg.Security.AdminCookieSecure, log))
		mux.HandleFunc("GET /v1/admin/auth/session", sessionStatus(auth))
		mux.Handle("/v1/admin/", requireSession(auth, admin, log))
	} else {
		mux.Handle("/v1/admin/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeError(w, http.StatusServiceUnavailable, "admin_auth_unavailable")
		}))
	}
	return recoverPanic(log, secureHeaders(cors(cfg.HTTP.AllowedOrigin, requestLog(log, metrics, mux))))
}

func createTrack(repo catalog.Repository, log *slog.Logger) http.HandlerFunc {
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
		if err == nil {
			log.Info("track_created", "track_id", t.PublicID, "title", t.Title, "artist", t.ArtistName)
		}
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
func deleteTrack(repo catalog.Repository, log *slog.Logger) http.HandlerFunc {
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
		if err == nil {
			log.Info("track_deleted", "track_id", id)
		}
		writeResult(w, track, err, http.StatusNoContent)
	}
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
func requestLog(log *slog.Logger, metrics *observability.Metrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		response := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(response, r)
		duration := time.Since(started)
		metrics.ObserveHTTP(duration)
		args := []any{"method", r.Method, "path", r.URL.Path, "status", response.status, "duration_ms", duration.Milliseconds(), "response_bytes", response.bytes}
		if requestID := response.Header().Get("X-Request-ID"); requestID != "" {
			args = append(args, "request_id", requestID)
		}
		switch {
		case response.status >= http.StatusInternalServerError:
			log.Error("http_request_completed", args...)
		case response.status >= http.StatusBadRequest:
			log.Warn("http_request_completed", args...)
		default:
			log.Debug("http_request_completed", args...)
		}
	})
}

type statusResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (w *statusResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *statusResponseWriter) Write(body []byte) (int, error) {
	n, err := w.ResponseWriter.Write(body)
	w.bytes += int64(n)
	return n, err
}
