# Lyra

Lyra is an audio-identification system with a Go backend and a static React/Vite client.

## Required reading

Before significant work, read `docs/STATUS.md`, `docs/ARCHITECTURE.md`, `docs/ALGORITHM.md`, and the active ExecPlan under `docs/exec-plans/`.

## Architectural invariants

- Lyra remains a modular monolith for v1. `web/` is a static React/Vite client, not a second backend or service.
- Exact fingerprint generation and matching remain in Go; Python/ML is out of scope.
- PostgreSQL is the source of truth; Valkey is only for jobs, cache, and rate limits.
- Query audio is never persisted. Fingerprint compatibility changes require a new version.
- Domain/DSP packages must not import HTTP or database infrastructure.
- Do not copy AGPL implementation source from reference projects.
- PostgreSQL and MinIO local data use named Docker volumes; do not remove volumes unless a destructive local reset is explicitly intended.

## Verification

Before ordinary implementation completion run `make verify`. Algorithm changes additionally require `make eval` and `make benchmark` when their corpus/index is available. Frontend changes additionally require `cd web && npm run lint && npm run build`.

## Local development

- Use Make targets for Go builds: `make build`, `make dev`, `make db-migrate`, and `make benchmark`.
- Never run bare `go build ./cmd/lyra`; it creates a generated `lyra` binary at the repository root. Make targets write the ignored binary to `bin/lyra`.
- `make dev` owns the local API, worker, Vite client, and Compose infrastructure. Ctrl-C is a successful graceful shutdown and stops all four; do not leave child processes running.
- Manually downloaded audio belongs in ignored `testdata/audio/` or `testdata/queries/`; never commit copyrighted or otherwise unlicensed audio.

## Go rules

Pass context at I/O boundaries; avoid mutable global application state and expected-error panics; wrap errors with context; use structured `slog`; never log secrets; prefer straightforward standard-library code and avoid speculative abstractions.

## Logging

- Use `slog` and stable event names with safe structured fields. Local text logs are colorized; production uses `LYRA_LOG_FORMAT=json`.
- Respect `LYRA_LOG_LEVEL` (`debug`, `info`, `warn`, `error`). Routine successful HTTP access logs are debug; meaningful product/lifecycle events are info.
- Never log raw query/reference audio, passwords, password hashes, cookies, session or CSRF values, authorization headers, API keys, object-store credentials, or database URLs. The redaction handler is a backstop, not permission to log sensitive data.

## Context persistence

After each milestone update `docs/STATUS.md` and the active ExecPlan with commands run, measured results only, issues, decisions, and the next milestone.
