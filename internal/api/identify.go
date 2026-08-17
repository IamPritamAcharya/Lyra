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
	"log/slog"
	"net/http"
	"os"
	"time"
)

type FileIdentifier interface {
	IdentifyFile(context.Context, string) (identify.Result, error)
}

func identifyHandler(maxBytes int64, identifier FileIdentifier, tracks catalog.Repository, metrics *observability.Metrics, log *slog.Logger) http.HandlerFunc {
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
			log.Error("identification_rejected", "request_id", requestID, "reason", "service_not_ready")
			writeError(w, http.StatusServiceUnavailable, "not_ready")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		if err := r.ParseMultipartForm(maxBytes); err != nil {
			log.Warn("identification_rejected", "request_id", requestID, "reason", "invalid_multipart")
			writeError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		file, _, err := r.FormFile("audio")
		if err != nil {
			log.Warn("identification_rejected", "request_id", requestID, "reason", "missing_audio")
			writeError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		defer file.Close()
		tmp, err := os.CreateTemp("", "lyra-query-*")
		if err != nil {
			log.Error("identification_failed", "request_id", requestID, "error", err)
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
				log.Error("identification_failed", "request_id", requestID, "reason", "catalog_lookup", "error", err)
				writeError(w, http.StatusServiceUnavailable, "catalog_unavailable")
				return
			}
			response["match"] = map[string]any{
				"track_id":            track.PublicID,
				"title":               track.Title,
				"artist":              track.ArtistName,
				"album":               track.AlbumName,
				"confidence":          result.Candidate.AlignmentCoherence,
				"match_strength":      "timing_aligned",
				"reference_offset_ms": result.Candidate.BestAlignmentOffset * fingerprint.HopSize * 1000 / fingerprint.SampleRate,
			}
		}
		matched = result.Matched
		if errors.Is(err, fingerprint.ErrInsufficientSignal) {
			response["reason"] = "insufficient_audio_signal"
		}
		if err != nil && !errors.Is(err, identify.ErrNoMatch) && !errors.Is(err, fingerprint.ErrInsufficientSignal) {
			log.Warn("identification_rejected", "request_id", requestID, "reason", "invalid_audio", "error", err)
			writeError(w, http.StatusBadRequest, "invalid_audio")
			return
		}
		fields := []any{"request_id", requestID, "matched", result.Matched, "candidates", len(result.Candidates), "duration_ms", time.Since(started).Milliseconds()}
		if result.Candidate != nil {
			fields = append(fields,
				"track_internal_id", result.Candidate.TrackID,
				"aligned_hits", result.Candidate.AlignedHits,
				"distinct_hashes", result.Candidate.UniqueAlignedHashes,
				"query_anchors", result.Candidate.UniqueQueryAnchors,
				"frequency_bins", result.Candidate.DistinctFrequencyBins,
				"alignment_span_frames", result.Candidate.AlignmentSpanFrames,
				"query_anchor_coverage", result.Candidate.QueryAnchorCoverage,
				"runner_up_alignment_hits", result.Candidate.RunnerUpAlignmentHits,
				"coherence", result.Candidate.AlignmentCoherence,
			)
		}
		if errors.Is(err, fingerprint.ErrInsufficientSignal) {
			fields = append(fields, "reason", "insufficient_audio_signal")
		} else if errors.Is(err, identify.ErrNoMatch) {
			fields = append(fields, "reason", "no_match")
		}
		log.Info("identification_completed", fields...)
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
