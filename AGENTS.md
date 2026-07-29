# Lyra

Lyra is a backend-only audio identification system.

## Required reading

Before significant work, read `docs/STATUS.md`, `docs/ARCHITECTURE.md`, `docs/ALGORITHM.md`, and the active ExecPlan under `docs/exec-plans/`.

## Architectural invariants

- Lyra remains a modular monolith for v1; no frontend or speculative services.
- Exact fingerprint generation and matching remain in Go; Python/ML is out of scope.
- PostgreSQL is the source of truth; Valkey is only for jobs, cache, and rate limits.
- Query audio is never persisted. Fingerprint compatibility changes require a new version.
- Domain/DSP packages must not import HTTP or database infrastructure.
- Do not copy AGPL implementation source from reference projects.

## Verification

Before ordinary implementation completion run `make verify`. Algorithm changes additionally require `make eval` and `make benchmark` when their corpus/index is available.

## Go rules

Pass context at I/O boundaries; avoid mutable global application state and expected-error panics; wrap errors with context; use structured `slog`; never log secrets; prefer straightforward standard-library code and avoid speculative abstractions.

## Context persistence

After each milestone update `docs/STATUS.md` and the active ExecPlan with commands run, measured results only, issues, decisions, and the next milestone.
