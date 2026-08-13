package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	authdomain "github.com/lyra/lyra/internal/auth"
)

const sessionCookie = "lyra_admin_session"

type AdminAuth interface {
	Login(context.Context, string, string) (string, string, time.Time, error)
	Validate(context.Context, string) error
	ValidateCSRF(context.Context, string, string) error
	Logout(context.Context, string) error
}

func login(auth AdminAuth, secure bool, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		token, csrf, expires, err := auth.Login(r.Context(), in.Username, in.Password)
		if errors.Is(err, authdomain.ErrInvalidCredentials) {
			log.Warn("admin_login_rejected", "reason", "invalid_credentials")
			writeError(w, http.StatusUnauthorized, "invalid_credentials")
			return
		}
		if err != nil {
			log.Error("admin_login_failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		log.Info("admin_login_succeeded", "username", in.Username)
		http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/v1/admin", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, Expires: expires, MaxAge: int(time.Until(expires).Seconds())})
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"username": in.Username, "csrf_token": csrf, "expires_at": expires}); err != nil {
			return
		}
	}
}

func sessionStatus(auth AdminAuth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := auth.Validate(r.Context(), sessionToken(r)); err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"authenticated": true})
	}
}
func logout(auth AdminAuth, secure bool, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = auth.Logout(r.Context(), sessionToken(r))
		log.Info("admin_logout_completed")
		http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/v1/admin", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
		w.WriteHeader(http.StatusNoContent)
	}
}
func requireSession(auth AdminAuth, next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := sessionToken(r)
		if err := auth.Validate(r.Context(), token); err != nil {
			log.Warn("admin_session_rejected", "path", r.URL.Path, "reason", "unauthenticated")
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			if err := auth.ValidateCSRF(r.Context(), token, r.Header.Get("X-CSRF-Token")); err != nil {
				log.Warn("admin_csrf_rejected", "path", r.URL.Path)
				writeError(w, http.StatusForbidden, "csrf_failed")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
func sessionToken(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return ""
	}
	return cookie.Value
}
