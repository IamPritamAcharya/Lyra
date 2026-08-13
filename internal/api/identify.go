package api

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lyra/lyra/internal/catalog"
	"github.com/lyra/lyra/internal/fingerprint"
	"github.com/lyra/lyra/internal/identify"
	"github.com/lyra/lyra/internal/observability"
	"net/http"
	"os"
	"time"
)

type FileIdentifier interface {
	IdentifyFile(context.Context, string) (identify.Result, error)
}

func identifyHandler(maxBytes int64, identifier FileIdentifier, tracks catalog.Repository, metrics *observability.Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		matched := false
		defer func() {
			if metrics != nil {
				metrics.ObserveIdentify(time.Since(started), matched)
			}
		}()
		requestID := newRequestID()
		w.Header().Set("X-Request-ID", requestID)
		if identifier == nil {
			writeError(w, http.StatusServiceUnavailable, "not_ready")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		if err := r.ParseMultipartForm(maxBytes); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		file, _, err := r.FormFile("audio")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		defer file.Close()
		tmp, err := os.CreateTemp("", "lyra-query-*")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		name := tmp.Name()
		defer os.Remove(name)
		if _, err := tmp.ReadFrom(file); err != nil {
			tmp.Close()
			writeError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		tmp.Close()
		result, err := identifier.IdentifyFile(r.Context(), name)
		response := map[string]any{"request_id": requestID, "matched": result.Matched, "match": nil, "processing_ms": time.Since(started).Milliseconds()}
		if result.Matched {
			track, err := tracks.GetByID(r.Context(), result.Candidate.TrackID)
			if err != nil {
				writeError(w, http.StatusServiceUnavailable, "catalog_unavailable")
				return
			}
			response["match"] = map[string]any{"track_id": track.PublicID, "title": track.Title, "artist": track.ArtistName, "album": track.AlbumName, "confidence": result.Candidate.AlignmentCoherence, "reference_offset_ms": result.Candidate.BestAlignmentOffset * fingerprint.HopSize * 1000 / fingerprint.SampleRate}
		}
		matched = result.Matched
		if errors.Is(err, fingerprint.ErrInsufficientSignal) {
			response["reason"] = "insufficient_audio_signal"
		}
		if err != nil && !errors.Is(err, identify.ErrNoMatch) && !errors.Is(err, fingerprint.ErrInsufficientSignal) {
			writeError(w, http.StatusBadRequest, "invalid_audio")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			return
		}
	}
}
func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "unavailable"
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}
