# Lyra v1 ExecPlan

## Purpose

Deliver a backend-only, deterministic acoustic-landmark music identification system as a Go modular monolith.

## Scope

Includes phases 0–11 from the bootstrap specification: DSP spike; production foundation; catalog/database; production fingerprinting; async ingestion; matcher; HTTP identify API; evaluation; measured scalability safeguards; production hardening; Docker deployment; release verification. Excludes humming, embeddings, ML, frontend, and distributed fingerprint storage.

## Progress

- [completed] Phase 0: repository context and deterministic DSP spike
- [completed] Phase 1: foundation
- [completed] Phase 2: schema/catalog
- [completed] Phase 3: production landmark-v1
- [completed] Phase 4: ingestion
- [completed] Phase 5: matcher
- [completed] Phase 6: identification API
- [in progress] Phase 7: evaluation
- [completed] Phase 8: scalability hardening
- [completed] Phase 9: production hardening
- [in progress] Phase 10: deployment
- [pending] Phase 11: release verification

## Architecture/Context

Read `AGENTS.md`, `docs/ARCHITECTURE.md`, and `docs/ALGORITHM.md` before changes. One binary supports all modes. PostgreSQL is truth; Valkey is queue/cache; private S3 objects retain only references. Query audio is ephemeral.

## Implementation steps

1. Build deterministic PCM/STFT/peak/landmark/hash extraction and golden tests.
2. Add validated configuration, lifecycle, HTTP health/metrics, Docker/Compose, CI.
3. Add migrations, sqlc query source, Postgres adapters, catalog admin API.
4. Add safe FFmpeg normalization and version-frozen production DSP.
5. Add S3 adapter, Asynq worker, lifecycle, atomic idempotent indexing.
6. Add batched lookup matcher and conservative no-match logic.
7. Add synchronous multipart identification API and contract.
8. Add legal reproducible evaluation and actual reporting.
9. Add measured query/index safeguards and synthetic benchmark.
10. Harden dependencies, auth, rate limiting, traces, shutdown, backup/recovery.
11. Run release gates and clean-environment smoke tests.

## Acceptance criteria

Each phase meets its bootstrap acceptance checks. v1 only completes after all release gates and documented smoke tests pass with real measured evaluation data.

## Verification commands

`make fmt`, `make lint`, `make vet`, `make vuln`, `make test`, `make test-race`, `make test-integration`, `make docker-build`, `make eval`, `make benchmark`, and `make verify` as applicable.

## Surprises & Discoveries

- Go toolchain initially available is 1.22.2; the environment did not complete Gonum/toolchain downloads. Phase 0 therefore uses a tested radix-2 FFT implementation; Phase 3 must switch to Gonum Fourier when dependency resolution is available.

## Decision Log

- landmark-v1 uses the frozen values in `docs/ALGORITHM.md`.
- Match-strength percentages must not be presented as probabilities. `timing_aligned` indicates that an accepted candidate passed temporal-evidence gates; Phase 7 measured data is required before adding calibrated strength levels or changing thresholds.

## Outcomes & Retrospective

Phase 0 completed with deterministic extraction, hash boundary/silence tests, and a committed synthetic golden vector (244 fingerprints; SHA-256 asserted). It was verified with `go test ./...`, `go vet ./...`, `make build`, and a local readiness smoke test.

Phase 2 has its initial migration/schema and a pgx catalog repository. The API integration and migration test remain open.

Phase 2 completed: migrations ran successfully against Compose PostgreSQL and the tagged lifecycle integration test passed. Catalog admin create/list/get/delete endpoints are present. Delete is synchronous only until Phase 4 introduces the deletion worker.

Phase 3 completed with safe no-shell FFmpeg/ffprobe execution, input validation, canonical PCM conversion, and Gonum Fourier. Phase 5 completed with a consumer-owned index contract, one batched hash lookup, offset-neighbourhood voting, deterministic ranking, and conservative seed rejection floors. Phase 4 ingestion remains required before release.

Reference upload endpoint wiring is now present: create track, post multipart reference audio, persist private object and metadata, then enqueue the indexing worker. A full local run awaits MinIO/Valkey image availability in this environment.

Phase 8 began with bounded matcher work: maximum query fingerprints, postings, and candidates are enforced, and the query hash set is sorted for deterministic lookups. A reproducible synthetic benchmark exists. On 2026-08-13, 1,000 synthetic tracks / 4,200 postings / 200 query fingerprints completed in 2 ms in memory and returned a match. This is not a PostgreSQL or audio-accuracy claim.

Hot-hash suppression now uses transactionally refreshed `fingerprint_hash_stats`; the PostgreSQL adapter filters hashes exceeding the configured maximum posting count prior to lookup. Unit tests verify suppression.

Phase 9 began by removing the in-memory serving fallback. `serve` requires configured PostgreSQL, S3-compatible object storage, and the admin key; it verifies the reference bucket at startup. This prevents a process that cannot perform correct identification from reporting ready.

Phase 9 adds panic recovery, restrictive HTTP security headers, and `scripts/backup.sh` with restore guidance in `docs/DEVELOPMENT.md`.

Phase 10 adds `render.yaml`, `docs/DEPLOYMENT.md`, and a real `make docker-build` target. Docker execution has not been verified in this session because the Docker socket is inaccessible to the execution sandbox.

Release preflight ran on 2026-08-13: race tests, vet, and golangci-lint passed. `govulncheck` could not retrieve `vuln.go.dev` due session network restrictions; Docker checks remain blocked by Docker socket access.

Matcher-scoring milestone on 2026-08-17: candidate evidence is now recomputed from the selected ±2-frame offset neighbourhood only. The matcher requires evidence across at least two anchor-frequency bins and rejects tied top-track aligned-hit scores. It records runner-up, query-coverage, span, and frequency-diversity diagnostics in safe identification logs. This is a scoring correction; `landmark-v1` extraction and hash compatibility are unchanged. The Identify UI now labels accepted evidence as `Timing aligned` instead of presenting temporal concentration as percentage confidence. Evaluation reports now include wrong-track matches, false negatives, false positives, and per-condition counts. `make verify`, `make test-race`, `web: npm run lint`, and `web: npm run build` passed. `make lint` could not run because `golangci-lint` is absent from the environment. `make benchmark` reported 1 ms for 1,000 synthetic tracks, 4,200 postings, and 200 query fingerprints; it is not a PostgreSQL or accuracy measurement. A legal corpus and `testdata/manifests/eval.json` are still absent, so `make eval` cannot run.

The user authorized a new frontend phase after the backend work. `web/` is a static React/TypeScript/Vite identification UI. ADR-001 records that Go remains the only backend. The public UI build passed with Node 18/Vite 5.

The frontend now includes the one permitted administrator workflow. `LYRA_ADMIN_USERNAME` and a bcrypt `LYRA_ADMIN_PASSWORD_HASH` configure the account. Successful login generates random opaque session and CSRF tokens; only SHA-256 token hashes are stored in PostgreSQL. The cookie is HttpOnly/SameSite=Lax (and configurable Secure), catalog writes require the CSRF header, sessions expire after 12 hours, and login has a per-IP limit. The static client receives no admin secret and keeps its CSRF token only in memory. `make dev` now starts local infrastructure, applies migrations, and runs API, worker, and Vite together. Go unit/vet and TypeScript lint/production-build checks passed; a real browser login smoke test remains pending.

Development observability now uses a colorized structured `slog` terminal handler; production can set `LYRA_LOG_FORMAT=json`. `LYRA_LOG_LEVEL` controls debug/info/warn/error filtering. A redaction handler protects fields with sensitive names, while service lifecycle, HTTP status/request IDs, authentication results, uploads, indexing jobs, fingerprint counts, and identification results are logged with safe fields. `make dev` starts processes in isolated process groups and tears down API, worker, Vite, and Compose infrastructure on Ctrl-C while preserving named Docker volumes. A Ctrl-C is reported as a successful exit. Make targets build to ignored `bin/lyra`, avoiding generated root binaries.

Brand/product-experience scope: establish a native SVG mark and lockup, document palette/tone/usage in `docs/BRAND.md`, apply the system to the responsive React Identify and Catalog views, and refresh the root README as an organization-facing project page. The work deliberately uses lightweight SVG/CSS rather than a generated raster mark or UI framework; it does not alter fingerprinting, API contracts, or auth behavior.
