package observability

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestLoggerRedactsSensitiveFields(t *testing.T) {
	var output bytes.Buffer
	log, err := NewLogger(&output, LoggingConfig{Level: "debug", Format: "json"})
	if err != nil {
		t.Fatal(err)
	}
	log.Info("test_event", "password", "plain-text", "csrf_token", "csrf", "track_id", "safe")
	got := output.String()
	for _, forbidden := range []string{"plain-text", `"csrf":"csrf"`} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("sensitive value leaked: %s", got)
		}
	}
	if !strings.Contains(got, "[REDACTED]") || !strings.Contains(got, "safe") {
		t.Fatalf("unexpected log: %s", got)
	}
}

func TestLoggerRejectsInvalidConfiguration(t *testing.T) {
	if _, err := NewLogger(&bytes.Buffer{}, LoggingConfig{Level: "loud", Format: "json"}); err == nil {
		t.Fatal("expected level validation error")
	}
	if _, err := NewLogger(&bytes.Buffer{}, LoggingConfig{Level: slog.LevelInfo.String(), Format: "xml"}); err == nil {
		t.Fatal("expected format validation error")
	}
}
