# Contributing to Lyra

Thanks for contributing. Lyra is a self-hostable, deterministic audio-identification system. Contributions should make the core recognition path more correct, measurable, secure, or maintainable—not introduce complexity for hypothetical scale.

## Before you start

Read [AGENTS.md](AGENTS.md), [docs/STATUS.md](docs/STATUS.md), [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md), and [docs/ALGORITHM.md](docs/ALGORITHM.md). For meaningful work, update the active plan in `docs/exec-plans/` as you go.

Open an issue or start a discussion before a large architectural change. New infrastructure, datastores, or network services require measured justification and an ADR.

## Local setup

Prerequisites: Go 1.22+, Docker Compose, FFmpeg, and Node 18+.

```bash
cp .env.example .env
cd web && npm install && cd ..
make dev
```

`make dev` starts local PostgreSQL, Valkey, MinIO, the Go API, worker, and Vite client. Press Ctrl-C to shut down all local processes and containers cleanly. Use Make targets rather than bare `go build ./cmd/lyra`; they write generated artifacts to ignored `bin/`.

## Development rules

- Keep Lyra a modular monolith. Do not add microservices, Kafka, Elasticsearch, vector search, ML, or a new datastore without an accepted ADR.
- Keep fingerprint/DSP code deterministic and independent of HTTP, PostgreSQL, and object storage.
- Treat `landmark-v1` as immutable. Any compatibility-affecting algorithm change requires a new fingerprint version, golden vectors, and updated algorithm documentation.
- PostgreSQL is the source of truth. Valkey is for jobs, rate limits, and short-lived coordination only.
- Query audio must never be retained. Reference audio remains private.
- Do not commit commercial, downloaded, or otherwise unlicensed audio. Use CC0/public-domain/synthetic material only for committed test data.
- Use `context.Context` at I/O boundaries, explicit errors, and structured `slog` events. Never log secrets, credentials, raw audio, cookies, or session/CSRF tokens.
- Keep the frontend lightweight: native SVG/CSS, no mandatory third-party font loading, and no blur-heavy or expensive visual effects.

## Verification

Run the checks relevant to your change before opening a pull request:

```bash
make fmt
make vet
make test
make test-race
make lint
make verify

cd web && npm run lint && npm run build
```

Algorithm changes must also run `make eval` and `make benchmark` once an appropriate legal corpus/index is available. Do not report performance or recognition results that have not been measured.

## Pull requests

Keep pull requests focused and explain:

1. What changes and why.
2. Any product, algorithm, security, migration, or operational impact.
3. Verification commands and actual results.
4. Documentation/contract changes, including OpenAPI where applicable.

Use clear commits. Do not include generated `bin/` files, `web/dist/`, local `.env` files, credentials, or manually downloaded test audio.

## Security issues

Do not open a public issue for a suspected vulnerability involving authentication, credentials, object access, query-audio retention, or arbitrary command execution. Contact the repository maintainer privately with reproduction details and impact.

## License

By contributing, you agree that your contributions are licensed under the [MIT License](LICENSE).
