# Lyra Status

Current phase: Frontend phase — public identification and protected admin UI (implemented; local browser smoke test pending).

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
- Panic recovery, restrictive security headers, and PostgreSQL backup/restore guidance are implemented.
- Render web/worker blueprint, deployment environment documentation, and functional `make docker-build` target are implemented.
- Static React/TypeScript/Vite public identification frontend is scaffolded and production-build verified.
- Single-admin browser authentication is implemented: bcrypt password verification, opaque server-side PostgreSQL sessions, HttpOnly cookies, CSRF checks for catalog writes, 12-hour expiry, and per-IP login rate limiting.
- Race tests, `go vet`, and golangci-lint have passed in the current environment after unchecked-error fixes.

Latest evaluation: not yet available; no recognition claims have been measured.

Known issues:
- Docker socket access is unavailable to this coding session, so the Docker image and local full-workflow smoke test cannot be rerun here. `govulncheck`/npm audit are blocked here because they cannot fetch remote vulnerability databases. A legal evaluation corpus remains unimplemented.

Next:
- Run `make dev` to apply migration `000002_admin_sessions` and complete the local browser login/catalog smoke test.

Last verification:
- `go test ./...`
- `go vet ./...`
- `go build ./cmd/lyra`
- local `GET /health/ready` smoke test
- `DATABASE_URL=... go test -tags=integration ./...` against local Compose PostgreSQL
- `./lyra benchmark --synthetic-tracks=1000` → 2 ms, 200 query fingerprints, 4,200 in-memory synthetic postings; matched=true.
- `go test -race ./...`, `go vet ./...`, and `golangci-lint run` passed on 2026-08-13.
- `go test ./...`, `go vet ./...`, and `web: npm run lint && npm run build` passed after the admin-auth implementation on 2026-08-13.
