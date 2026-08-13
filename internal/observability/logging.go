package observability

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
)

// LoggingConfig controls the process-wide structured logger. Text is intended
// for local terminals; JSON is intended for production log collection.
type LoggingConfig struct {
	Level  string
	Format string
}

func NewLogger(out io.Writer, cfg LoggingConfig) (*slog.Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}
	options := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	switch cfg.Format {
	case "text":
		handler = newTerminalHandler(out, options)
	case "json":
		handler = slog.NewJSONHandler(out, options)
	default:
		return nil, fmt.Errorf("invalid log format %q", cfg.Format)
	}
	return slog.New(redactingHandler{next: handler}), nil
}

func parseLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(raw) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid log level %q", raw)
	}
}

// terminalHandler deliberately keeps development output compact while retaining
// the same structured fields emitted as JSON in production.
type terminalHandler struct {
	out    io.Writer
	level  slog.Leveler
	attrs  []slog.Attr
	groups []string
	mu     *sync.Mutex
}

func newTerminalHandler(out io.Writer, options *slog.HandlerOptions) slog.Handler {
	return &terminalHandler{out: out, level: options.Level, mu: &sync.Mutex{}}
}

func (h *terminalHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *terminalHandler) Handle(_ context.Context, record slog.Record) error {
	attrs := append([]slog.Attr{}, h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})
	icon, category, message, color := friendlyEvent(record.Message, record.Level)
	header := fmt.Sprintf("\x1b[2m%s\x1b[0m  %s%s\x1b[0m  \x1b[1m%-9s\x1b[0m %s", record.Time.Format("15:04:05.000"), color, icon, category, message)
	parts := make([]string, 0, len(attrs))
	for _, attr := range attrs {
		parts = append(parts, formatAttr(h.groups, attr))
	}
	line := header
	for i := 0; i < len(parts); i += 4 {
		end := i + 4
		if end > len(parts) {
			end = len(parts)
		}
		line += "\n    \x1b[2m•\x1b[0m " + strings.Join(parts[i:end], "  \x1b[2m•\x1b[0m ")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := fmt.Fprintln(h.out, line)
	return err
}

func (h *terminalHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &clone
}

func (h *terminalHandler) WithGroup(name string) slog.Handler {
	clone := *h
	clone.groups = append(append([]string{}, h.groups...), name)
	return &clone
}

func levelColor(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "\x1b[31m"
	case level >= slog.LevelWarn:
		return "\x1b[33m"
	case level >= slog.LevelInfo:
		return "\x1b[32m"
	default:
		return "\x1b[36m"
	}
}

func friendlyEvent(event string, level slog.Level) (icon, category, message, color string) {
	color = levelColor(level)
	category, message = "LYRA", strings.ReplaceAll(event, "_", " ")
	switch event {
	case "server_started":
		return "●", "SERVER", "API is ready", "\x1b[32m"
	case "server_starting":
		return "…", "SERVER", "Starting API", "\x1b[36m"
	case "server_shutdown_started", "server_shutdown_completed":
		return "◌", "SERVER", strings.ReplaceAll(event, "server_", ""), "\x1b[33m"
	case "worker_started":
		return "●", "WORKER", "Ready for indexing jobs", "\x1b[32m"
	case "worker_starting":
		return "…", "WORKER", "Starting worker", "\x1b[36m"
	case "worker_shutdown_started", "worker_shutdown_completed":
		return "◌", "WORKER", strings.ReplaceAll(event, "worker_", ""), "\x1b[33m"
	case "admin_login_succeeded":
		return "✓", "AUTH", "Admin signed in", "\x1b[32m"
	case "admin_logout_completed":
		return "✓", "AUTH", "Admin signed out", "\x1b[32m"
	case "admin_login_rejected", "admin_session_rejected", "admin_csrf_rejected":
		return "!", "AUTH", strings.ReplaceAll(event, "admin_", ""), "\x1b[33m"
	case "track_created":
		return "+", "CATALOG", "Track created", "\x1b[32m"
	case "track_deleted":
		return "−", "CATALOG", "Track deleted", "\x1b[33m"
	case "reference_upload_accepted", "reference_audio_stored":
		return "↑", "UPLOAD", "Reference audio accepted", "\x1b[32m"
	case "fingerprint_job_enqueued":
		return "→", "INDEX", "Fingerprint job queued", "\x1b[36m"
	case "fingerprint_job_started":
		return "…", "INDEX", "Fingerprinting track", "\x1b[36m"
	case "track_indexed":
		return "✓", "INDEX", "Track is ready", "\x1b[32m"
	case "fingerprint_job_failed":
		return "×", "INDEX", "Fingerprinting failed", "\x1b[31m"
	case "identification_completed":
		return "♫", "MATCH", "Identification complete", "\x1b[35m"
	case "identification_rejected", "identification_failed":
		return "!", "MATCH", strings.ReplaceAll(event, "identification_", ""), color
	case "http_request_completed":
		return "↔", "HTTP", "Request complete", color
	}
	return "•", category, message, color
}

func formatAttr(groups []string, attr slog.Attr) string {
	key := strings.Join(append(append([]string{}, groups...), attr.Key), ".")
	value := attr.Value.Resolve()
	if key == "duration_ms" {
		return "duration=" + fmt.Sprint(value.Any()) + "ms"
	}
	if key == "coherence" {
		if coherence, ok := value.Any().(float64); ok {
			return fmt.Sprintf("confidence=%.1f%%", coherence*100)
		}
	}
	if key == "matched" {
		return "matched=" + fmt.Sprint(value.Any())
	}
	if value.Kind() == slog.KindString {
		return key + "=" + fmt.Sprintf("%q", value.String())
	}
	return key + "=" + fmt.Sprint(value.Any())
}

type redactingHandler struct{ next slog.Handler }

func (h redactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}
func (h redactingHandler) Handle(ctx context.Context, record slog.Record) error {
	clean := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	record.Attrs(func(attr slog.Attr) bool {
		clean.AddAttrs(redactAttr(attr))
		return true
	})
	return h.next.Handle(ctx, clean)
}
func (h redactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return redactingHandler{next: h.next.WithAttrs(redactAttrs(attrs))}
}
func (h redactingHandler) WithGroup(name string) slog.Handler {
	return redactingHandler{next: h.next.WithGroup(name)}
}

func redactAttrs(attrs []slog.Attr) []slog.Attr {
	clean := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		clean = append(clean, redactAttr(attr))
	}
	return clean
}

func redactAttr(attr slog.Attr) slog.Attr {
	if sensitiveKey(attr.Key) {
		return slog.String(attr.Key, "[REDACTED]")
	}
	if attr.Value.Kind() != slog.KindGroup {
		return attr
	}
	return slog.Attr{Key: attr.Key, Value: slog.GroupValue(redactAttrs(attr.Value.Group())...)}
}

func sensitiveKey(key string) bool {
	key = strings.ToLower(key)
	for _, part := range []string{"password", "secret", "token", "cookie", "csrf", "authorization", "credential", "api_key"} {
		if strings.Contains(key, part) {
			return true
		}
	}
	return false
}
