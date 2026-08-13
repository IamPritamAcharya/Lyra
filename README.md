# Lyra

Lyra is a Go service that identifies short recordings of audio previously indexed as reference material. It uses deterministic acoustic landmarks, an inverted PostgreSQL fingerprint index, and temporal-offset voting—no machine learning.

## Current capabilities

- Admin track catalog and private reference-audio upload
- Asynchronous fingerprint indexing with PostgreSQL, Valkey, and S3-compatible storage
- Safe FFprobe/FFmpeg canonicalization and deterministic landmark-v1 extraction
- Multipart `POST /v1/identify` matching, no-match and insufficient-signal handling
- Admin UI with a single bcrypt-protected, server-side session account
- Health endpoints, migrations, worker mode, OpenAPI contract, and importable Postman collection

See [docs/STATUS.md](docs/STATUS.md) for the precise implementation state and unmeasured limitations.

## Run locally

Prerequisites: Go 1.22+, Docker/Compose, and FFmpeg.

```bash
cp .env.example .env
# Set LYRA_ADMIN_PASSWORD_HASH as described in docs/DEVELOPMENT.md.
cd web && npm install && cd ..
make dev
```

`make dev` starts PostgreSQL, Valkey, MinIO, waits until they are ready, runs migrations, then starts the Go API, worker, and Vite frontend. Press `Ctrl-C` to gracefully stop the API, worker, frontend, and local Docker infrastructure.

An intentional `Ctrl-C` is a successful shutdown and exits `make dev` without an error. Local builds are written to ignored `bin/lyra`; the project root stays free of generated binaries.

Local PostgreSQL and MinIO use named Docker volumes, so `make infra-down` does not erase tracks or uploaded reference audio. Removing the Docker volumes is a deliberate destructive reset.

## Logging

Local development defaults to colorized structured text logs. Set `LYRA_LOG_LEVEL` to `debug`, `info`, `warn`, or `error`; set `LYRA_LOG_FORMAT=json` for production log collection. Logs include request IDs, lifecycle events, indexing/fingerprint counts, and safe matching statistics. Passwords, secrets, cookies, CSRF values, session tokens, and query audio are never logged.

Verify readiness:

```bash
curl http://localhost:8080/health/ready
```

## Test end-to-end with Postman

Import [postman/Lyra.postman_collection.json](postman/Lyra.postman_collection.json) and [postman/Lyra.local.postman_environment.json](postman/Lyra.local.postman_environment.json). Select **Lyra Local**, then run:

1. `Health / Ready`
2. `Admin tracks / Create track (stores track_id)`
3. `Admin tracks / Upload reference audio (queues indexing)`
4. `Admin tracks / Get track / polling status` until `Status` is `READY`
5. `Identification / Identify indexed query clip`

Run `Admin auth / Login` first; it stores the CSRF token, while Postman retains the HttpOnly session cookie. The collection setup and audio path variables are described in [postman/README.md](postman/README.md).

## Commands

```bash
make build
make test
make test-race
make vet
make verify
make infra-up
make infra-down
make dev
make db-migrate
```

The binary supports `serve`, `worker`, `migrate`, `fingerprint`, and `eval` modes.

## Deployment

Use the same image for the API (`lyra serve`) and worker (`lyra worker`). The required environment variables, migration sequence, backup guidance, and Render blueprint are documented in [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).

## Frontend

The static React/Vite UI under [`web/`](web/) includes public identification and the single admin catalog workflow. Run `cd web && npm install && npm run dev`; see [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md#frontend).

## License

License selection is pending.
