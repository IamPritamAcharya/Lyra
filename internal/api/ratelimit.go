package api

import (
	"golang.org/x/time/rate"
	"net"
	"net/http"
	"sync"
)

type limiter struct {
	mu    sync.Mutex
	perIP map[string]*rate.Limiter
	limit rate.Limit
}

func newLimiter(perMinute int) *limiter {
	return &limiter{perIP: map[string]*rate.Limiter{}, limit: rate.Limit(float64(perMinute) / 60)}
}
func (l *limiter) allow(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	v := l.perIP[host]
	if v == nil {
		v = rate.NewLimiter(l.limit, 1)
		l.perIP[host] = v
	}
	return v.Allow()
}
func (l *limiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(r) {
			writeError(w, http.StatusTooManyRequests, "rate_limited")
			return
		}
		next.ServeHTTP(w, r)
	})
}
