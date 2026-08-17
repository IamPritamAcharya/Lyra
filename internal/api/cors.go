package api

import (
	"net/http"
	"strings"
)

// cors permits only an explicitly configured browser origin.
func cors(origins string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestOrigin := r.Header.Get("Origin")
		if allowedOrigin(origins, requestOrigin) {
			w.Header().Set("Access-Control-Allow-Origin", requestOrigin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token, X-Lyra-Live-Capture-Ms")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func allowedOrigin(origins, requestOrigin string) bool {
	if requestOrigin == "" {
		return false
	}
	for _, origin := range strings.Split(origins, ",") {
		if requestOrigin == strings.TrimSpace(origin) {
			return true
		}
	}
	return false
}
