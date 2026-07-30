# Lyra

Lyra is a backend-only, deterministic acoustic-landmark music identification system. It is a Go modular monolith designed to identify short recordings of audio that has been indexed as reference material.

Current implementation status and known limitations are in [docs/STATUS.md](docs/STATUS.md). Start locally with `cp .env.example .env` and `make dev`; run the DSP spike with `go run ./cmd/lyra fingerprint canonical.wav`.
