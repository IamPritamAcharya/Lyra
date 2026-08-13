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
	parts := []string{record.Time.Format("15:04:05.000"), colorLevel(record.Level), record.Message}
	for _, attr := range attrs {
		parts = append(parts, formatAttr(h.groups, attr))
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := fmt.Fprintln(h.out, strings.Join(parts, " "))
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

func colorLevel(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "\x1b[31mERROR\x1b[0m"
	case level >= slog.LevelWarn:
		return "\x1b[33mWARN\x1b[0m"
	case level >= slog.LevelInfo:
		return "\x1b[32mINFO\x1b[0m"
	default:
		return "\x1b[36mDEBUG\x1b[0m"
	}
}

func formatAttr(groups []string, attr slog.Attr) string {
	key := strings.Join(append(append([]string{}, groups...), attr.Key), ".")
	value := attr.Value.Resolve()
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
