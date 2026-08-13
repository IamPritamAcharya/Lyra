package api

import (
	"bytes"
	"context"
	"github.com/lyra/lyra/internal/catalog"
	"github.com/lyra/lyra/internal/config"
	"github.com/lyra/lyra/internal/identify"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubIdentifier struct {
	result identify.Result
	err    error
}

func (s stubIdentifier) IdentifyFile(context.Context, string) (identify.Result, error) {
	return s.result, s.err
}
func TestIdentifyReturnsPublicMetadataAndRemovesTemporaryInput(t *testing.T) {
	repo := catalog.NewMemoryRepository()
	track, err := repo.Create(context.Background(), catalog.CreateTrack{Title: "title", ArtistName: "artist"})
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("audio", "query.wav")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("temporary query audio")); err != nil {
		t.Fatal(err)
	}
	form.Close()
	handler := New(config.Config{Security: config.SecurityConfig{MaxIdentifyBytes: 1024}}, slog.New(slog.NewTextHandler(io.Discard, nil)), repo, func(*http.Request) error { return nil }, stubIdentifier{result: identify.Result{Matched: true, Candidate: &identify.Candidate{TrackID: track.ID, AlignmentCoherence: .75, BestAlignmentOffset: 12}}}, nil)
	request := httptest.NewRequest(http.MethodPost, "/v1/identify", &body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("X-Request-ID") == "" {
		t.Fatal("request id missing")
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(track.PublicID)) || bytes.Contains(response.Body.Bytes(), []byte("track_internal_id")) {
		t.Fatalf("body=%s", response.Body.String())
	}
}
