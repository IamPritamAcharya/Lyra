# Lyra Status

Current phase: Phase 9 — production hardening (in progress).

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
- Phase 4 worker execution mode and Phase 5 matcher are implemented; Phase 6 multipart identify handler is wired to the matcher.
- Phase 6 completed with ephemeral multipart handling, public metadata responses, request IDs, request limits, and per-process identify rate limiting.
- Synthetic matcher benchmark command implemented. Initial in-memory 1,000-track baseline completed; see `docs/BENCHMARKS.md`.
- Prometheus-compatible HTTP and identification counters/latency summaries replace the metrics placeholder.
- Transactionally refreshed hash posting statistics and hot-hash suppression are implemented before PostgreSQL fingerprint lookup.
- Serving now fails fast when PostgreSQL, object storage, or required security configuration is absent instead of exposing an in-memory fallback.

Latest evaluation: not yet available; no recognition claims have been measured.

Known issues:
- Docker socket access is unavailable to this coding session, so the local full-workflow smoke test cannot be rerun here. A legal evaluation corpus remains unimplemented.

Next:
- Add tracing, backup/restore guidance, and failure/recovery tests. Generate/run the legal evaluation corpus when local dependencies are available.

Last verification:
- `go test ./...`
- `go vet ./...`
- `go build ./cmd/lyra`
- local `GET /health/ready` smoke test
- `DATABASE_URL=... go test -tags=integration ./...` against local Compose PostgreSQL
- `./lyra benchmark --synthetic-tracks=1000` → 2 ms, 200 query fingerprints, 4,200 in-memory synthetic postings; matched=true.
