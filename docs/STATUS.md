# Lyra Status

Current phase: Phase 0 — repository and algorithm spike (in progress).

Completed:
- Persistent project context and repository skeleton created.
- Canonical PCM16 mono WAV reader and deterministic landmark-v1 extractor implemented.
- Hash packing boundary tests, deterministic extraction test, and silence rejection test implemented.
- Minimal `serve` command, graceful shutdown, health endpoints, and metrics shell implemented.

Latest evaluation: not yet available; no recognition claims have been measured.

Known issues:
- A checked-in binary WAV golden fixture is still required to complete Phase 0 acceptance.
- The environment could not complete Gonum/toolchain downloads; the current phase-0 FFT is a tested local radix-2 implementation. Production Phase 3 must adopt Gonum Fourier once dependency resolution is available.
- Database, object storage, queue, ingestion, matcher, identify API, evaluation corpus, Docker, and CI are not yet implemented.

Next:
- Add legal synthetic WAV golden data, then proceed to production foundation and schema/catalog.

Last verification:
- `go test ./...`
- `go vet ./...`
- `go build ./cmd/lyra`
- local `GET /health/ready` smoke test
