# Lyra Status

Current phase: Phase 3 — production landmark-v1 (in progress).

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

Latest evaluation: not yet available; no recognition claims have been measured.

Known issues:
- The environment could not complete Gonum/toolchain downloads; the current phase-0 FFT is a tested local radix-2 implementation. Production Phase 3 must adopt Gonum Fourier once dependency resolution is available.
- PostgreSQL adapter generation/runtime integration, object storage, queue, ingestion, matcher, identify API, and evaluation corpus are not yet implemented.

Next:
- Implement safe FFmpeg probing/normalization and production audio validation.

Last verification:
- `go test ./...`
- `go vet ./...`
- `go build ./cmd/lyra`
- local `GET /health/ready` smoke test
- `DATABASE_URL=... go test -tags=integration ./...` against local Compose PostgreSQL
