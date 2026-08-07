package api

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/lyra/lyra/internal/fingerprint"
	"github.com/lyra/lyra/internal/identify"
	"net/http"
	"os"
	"time"
)

type FileIdentifier interface {
	IdentifyFile(context.Context, string) (identify.Result, error)
}

func identifyHandler(maxBytes int64, identifier FileIdentifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
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
		response := map[string]any{"matched": result.Matched, "match": nil, "processing_ms": time.Since(started).Milliseconds()}
		if result.Matched {
			response["match"] = map[string]any{"track_internal_id": result.Candidate.TrackID, "confidence": result.Candidate.AlignmentCoherence, "reference_offset_frames": result.Candidate.BestAlignmentOffset}
		}
		if errors.Is(err, fingerprint.ErrInsufficientSignal) {
			response["reason"] = "insufficient_audio_signal"
		}
		if err != nil && !errors.Is(err, identify.ErrNoMatch) && !errors.Is(err, fingerprint.ErrInsufficientSignal) {
			writeError(w, http.StatusBadRequest, "invalid_audio")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
