# Lyra v1 ExecPlan

## Purpose

Deliver a backend-only, deterministic acoustic-landmark music identification system as a Go modular monolith.

## Scope

Includes phases 0–11 from the bootstrap specification: DSP spike; production foundation; catalog/database; production fingerprinting; async ingestion; matcher; HTTP identify API; evaluation; measured scalability safeguards; production hardening; Docker deployment; release verification. Excludes humming, embeddings, ML, frontend, and distributed fingerprint storage.

## Progress

- [completed] Phase 0: repository context and deterministic DSP spike
- [completed] Phase 1: foundation
- [in progress] Phase 2: schema/catalog
- [pending] Phase 3: production landmark-v1
- [pending] Phase 4: ingestion
- [pending] Phase 5: matcher
- [pending] Phase 6: identification API
- [pending] Phase 7: evaluation
- [pending] Phase 8: scalability hardening
- [pending] Phase 9: production hardening
- [pending] Phase 10: deployment
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

## Outcomes & Retrospective

Phase 0 completed with deterministic extraction, hash boundary/silence tests, and a committed synthetic golden vector (244 fingerprints; SHA-256 asserted). It was verified with `go test ./...`, `go vet ./...`, `go build ./cmd/lyra`, and a local readiness smoke test.
