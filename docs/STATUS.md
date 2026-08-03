# Lyra Status

Current phase: Phase 4 — ingestion (in progress).

Completed:
- Persistent project context and repository skeleton created.
- Canonical PCM16 mono WAV reader and deterministic landmark-v1 extractor implemented.
- Hash packing boundary tests, deterministic extraction test, and silence rejection test implemented.
- A synthetic deterministic landmark-v1 golden vector (244 packed fingerprints, SHA-256 asserted in tests) is committed.
- Minimal `serve` command, graceful shutdown, health endpoints, and metrics shell implemented.
- Docker/Compose and baseline GitHub Actions workflow created.
- PostgreSQL migration pair, sqlc query definitions, and explicit track lifecycle state machine created.
- pgx/PostgreSQL catalog repository added, including locked transactional lifecycle transitions.
- Admin catalog create/list/get HTTP routes with timing-safe key comparison and contract skeleton added.
- `lyra migrate` now applies golang-migrate migrations; it was executed against local Compose PostgreSQL and created all five expected tables.
- Tagged PostgreSQL catalog lifecycle integration test passed against Compose PostgreSQL.
- Phase 3 completed: safe FFmpeg/ffprobe boundary validates source audio and produces canonical mono 11025 Hz PCM16; DSP now uses Gonum Fourier.
- Deterministic matcher with one batched inverted-index lookup, ±2-frame offset voting, ranking, and no-match/insufficient-signal decisions implemented and unit-tested.

Latest evaluation: not yet available; no recognition claims have been measured.

Known issues:
- Object storage, queue, asynchronous ingestion, identification API, and evaluation corpus are not yet implemented.

Next:
- Implement reference ingestion and atomic fingerprint persistence (Phase 4), then wire CLI/HTTP identification.

Last verification:
- `go test ./...`
- `go vet ./...`
- `go build ./cmd/lyra`
- local `GET /health/ready` smoke test
- `DATABASE_URL=... go test -tags=integration ./...` against local Compose PostgreSQL
