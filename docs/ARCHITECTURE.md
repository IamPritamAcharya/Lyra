# Architecture

Lyra v1 is one Go modular-monolith binary. `serve`, `worker`, `migrate`, `ingest`, `identify`, `eval`, and `benchmark` are execution modes, not services. PostgreSQL holds catalog and fingerprint truth; Valkey carries Asynq work; private S3-compatible storage holds reference audio. Query recordings use controlled temporary files and are deleted after synchronous identification.

The optional `web/` React/Vite static frontend is not another backend. It calls the Go API directly; browser code never contains an admin secret. Its CORS origins are explicitly restricted by the comma-separated `LYRA_ALLOWED_ORIGIN` allowlist.

During temporary phone-tunnel testing, the local Vite page continues to use the local API at `http://localhost:8080`; only a non-local public frontend uses `VITE_LYRA_API_BASE_URL`. This keeps the local browser’s development session cookie same-site while the phone uses HTTPS.

The public Identify view can capture microphone audio with `getUserMedia` and the Web Audio API. Capture is held in browser memory only and each checkpoint is encoded as a complete PCM WAV file before submission to the existing synchronous identify endpoint at five, eight, and eleven seconds; it stops immediately on an accepted match or at 15 seconds, then makes one final check if necessary. The API writes a controlled temporary query file only for processing and deletes it before responding; microphone/query audio is never object-stored or added to the catalog.

Live identify requests include only their elapsed capture duration in a bounded `X-Lyra-Live-Capture-Ms` header. The API validates the 1–15,000 ms value and records it as the safe `live_capture_ms` field in the existing structured identification event, alongside match result, reason, and processing duration.

There is one configured administrator, not a user account system. The administrator password is supplied as a bcrypt hash. Login creates a random opaque server-side session whose token hash and CSRF-token hash are stored in PostgreSQL. The browser receives only an HttpOnly cookie plus a CSRF token held in memory; every catalog write verifies both. Sessions expire after 12 hours. Local HTTP development may set `LYRA_ADMIN_COOKIE_SECURE=false`; deployed HTTPS environments must set it true.

Dependencies point inward: API handlers call application use cases; catalog, ingest, and identify depend on small consumer-owned interfaces; audio and fingerprint packages are deterministic and infrastructure-free. PostgreSQL will initially implement the fingerprint inverted index behind `FingerprintIndex`.

Track lifecycle is `CREATED -> UPLOADED -> INDEXING -> READY`, with `FAILED`, `REINDEXING`, and `DELETING -> DELETED` explicit state transitions. Index replacement is transactional: generate outside a transaction, then delete/version-insert/status-update in one transaction.

A `CREATED` track may also transition to `DELETING` so an abandoned metadata record can be removed before reference audio exists.
