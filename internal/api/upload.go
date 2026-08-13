package api

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"
)

type ReferenceUploader interface {
	Upload(context.Context, string, string, string, io.Reader, int64, string) error
}

func uploadTrackAudio(u ReferenceUploader, maxBytes int64, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if u == nil {
			writeError(w, http.StatusServiceUnavailable, "not_ready")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		if err := r.ParseMultipartForm(maxBytes); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		file, header, err := r.FormFile("audio")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		defer file.Close()
		if header.Size <= 0 {
			writeError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		key := fmt.Sprintf("reference/%s/%d%s", r.PathValue("id"), time.Now().UnixNano(), filepath.Ext(header.Filename))
		if err := u.Upload(r.Context(), r.PathValue("id"), key, header.Filename, file, header.Size, header.Header.Get("Content-Type")); err != nil {
			log.Warn("reference_upload_rejected", "track_id", r.PathValue("id"), "size_bytes", header.Size, "error", err)
			writeError(w, http.StatusBadRequest, "upload_failed")
			return
		}
		log.Info("reference_upload_accepted", "track_id", r.PathValue("id"), "size_bytes", header.Size, "mime_type", header.Header.Get("Content-Type"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"uploaded"}`))
	}
}
