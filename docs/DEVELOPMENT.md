# Development and operations

## Frontend

The public browser interface lives in `web/`. Its Node dependencies use Vite 5 for compatibility with Node 18; use Node 20.19+ before upgrading to newer Vite versions.

```bash
cd web
cp .env.example .env
npm install
cd ..
make dev
```

`make dev` starts local PostgreSQL, Valkey, MinIO, waits until they are ready, runs migrations, and starts the Go API, worker, and Vite at `http://localhost:5173`. Open exactly that `localhost` URL: the API's development CORS policy intentionally permits it, not `127.0.0.1`. Press `Ctrl-C` to gracefully stop the API, worker, frontend, and the Docker infrastructure. The frontend’s `VITE_LYRA_API_BASE_URL` must point to the API (default `http://localhost:8080`).

`Ctrl-C` is an intentional successful shutdown, so `make dev` exits with code 0 after cleanup. `make build`, `make db-migrate`, `make eval`, and `make benchmark` write/use the ignored `bin/lyra` binary; do not run bare `go build ./cmd/lyra`, which makes Go place a `lyra` executable at the repository root.

PostgreSQL and MinIO are backed by named Docker volumes. `make infra-down` stops containers without deleting indexed tracks or reference audio. Only remove those Docker volumes when you deliberately want an empty local Lyra installation.

## Database viewer

Adminer is included only in the local Compose profile. `make dev` and `make infra-up` expose it at [http://localhost:8081](http://localhost:8081). Use:

```text
System: PostgreSQL
Server: postgres
Username: lyra
Password: lyra
Database: lyra
```

Adminer is for inspecting local PostgreSQL tables and running development SQL. It is not part of Lyra’s application image or production deployment. Do not expose it publicly.

## Logging

`LYRA_LOG_FORMAT=text` emits colorized terminal logs for local development. `LYRA_LOG_FORMAT=json` emits structured JSON for production collectors. Set `LYRA_LOG_LEVEL` to `debug`, `info`, `warn`, or `error`; production should normally use `info` or `warn` with JSON.

Every operational event uses a stable event name such as `track_indexed`, `identification_completed`, or `admin_login_rejected`. Useful safe fields include request ID, public track ID, status, fingerprint count, candidate count, match statistics, HTTP status, and duration. Never add raw query audio, file content, passwords, password hashes, session IDs, cookies, CSRF tokens, API keys, or storage credentials to a log field. The logger redacts common sensitive field names as an additional safeguard.

The browser admin is a single configured account. Generate its bcrypt password hash once, put only that hash in `.env`, and keep the plain-text password out of the repository:

```bash
htpasswd -bnBC 12 "" 'choose-a-long-unique-password' | tr -d ':\n'
```

Set the result as `LYRA_ADMIN_PASSWORD_HASH`; optionally change `LYRA_ADMIN_USERNAME`. `LYRA_ADMIN_COOKIE_SECURE=false` is for local HTTP only. Set it to `true` for HTTPS deployments. The browser receives an HttpOnly session cookie and an in-memory CSRF token—not an admin secret.

## Manual audio checks

Use `testdata/audio/` for a full local reference recording and `testdata/queries/` for a short excerpt from that exact recording. Those directories are intentionally ignored by Git so manually downloaded or copyrighted audio cannot be committed. Upload the full file through the Admin UI, wait for `READY`, then submit its excerpt through the Identify UI. For a dependable initial check, use a 5–10 second excerpt cut directly from the exact uploaded file.

## Backups

PostgreSQL stores Lyra's catalog and fingerprint source of truth. Create a custom-format backup with:

```bash
export DATABASE_URL='postgres://…'
export LYRA_BACKUP_DIR="$PWD/backups"
./scripts/backup.sh
```

Restore into a prepared database with `pg_restore --clean --if-exists --dbname "$DATABASE_URL" path/to/lyra-*.dump`. Back up the private S3/MinIO reference-audio bucket under a separate object-storage policy.
