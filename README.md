# Lyra

Lyra is a backend-only Go service that identifies short recordings of audio previously indexed as reference material. It uses deterministic acoustic landmarks, an inverted PostgreSQL fingerprint index, and temporal-offset voting—no machine learning or frontend.

## Current capabilities

- Admin track catalog and private reference-audio upload
- Asynchronous fingerprint indexing with PostgreSQL, Valkey, and S3-compatible storage
- Safe FFprobe/FFmpeg canonicalization and deterministic landmark-v1 extraction
- Multipart `POST /v1/identify` matching, no-match and insufficient-signal handling
- Health endpoints, migrations, worker mode, OpenAPI contract, and importable Postman collection

See [docs/STATUS.md](docs/STATUS.md) for the precise implementation state and unmeasured limitations.

## Run locally

Prerequisites: Go 1.22+, Docker/Compose, and FFmpeg.

```bash
cp .env.example .env
make infra-up

set -a && source .env && set +a
make db-migrate
```

In separate terminals, after exporting the same `.env` values:

```bash
go run ./cmd/lyra serve
go run ./cmd/lyra worker
```

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

The collection setup and audio path variables are described in [postman/README.md](postman/README.md).

## Commands

```bash
make build
make test
make test-race
make vet
make verify
make infra-up
make infra-down
make db-migrate
```

The binary supports `serve`, `worker`, `migrate`, `fingerprint`, and `eval` modes.

## Deployment

Use the same image for the API (`lyra serve`) and worker (`lyra worker`). The required environment variables, migration sequence, backup guidance, and Render blueprint are documented in [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).

## License

License selection is pending.
