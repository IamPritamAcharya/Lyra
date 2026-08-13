# Deployment

Lyra deploys one Docker image in separate execution modes:

```text
web:    lyra serve
worker: lyra worker
release: lyra migrate
```

Run migrations exactly once per deployment before updating the web and worker processes:

```bash
lyra migrate
```

## Required environment

| Variable | Purpose |
| --- | --- |
| `DATABASE_URL` | PostgreSQL connection string |
| `REDIS_ADDR` | Valkey/Redis address used by Asynq |
| `S3_ENDPOINT` | Private S3-compatible storage endpoint |
| `S3_ACCESS_KEY`, `S3_SECRET_KEY` | Object storage credentials |
| `S3_BUCKET` | Reference-audio bucket |
| `S3_SECURE` | `true` for HTTPS object storage |
| `LYRA_ADMIN_USERNAME` | Single configured browser-admin username |
| `LYRA_ADMIN_PASSWORD_HASH` | bcrypt hash for that admin password; never use the plain-text password here |
| `LYRA_ADMIN_COOKIE_SECURE` | Must be `true` for an HTTPS deployment |
| `LYRA_ALLOWED_ORIGIN` | Exact deployed frontend origin allowed to make browser API requests |

Use a private S3/R2/MinIO bucket. Query recordings are never object-stored; only uploaded reference audio is retained.

## Render

`render.yaml` defines one web service and one worker. Create managed PostgreSQL, a Redis-compatible service, and S3/R2 storage separately, then fill the environment variables in Render. Execute `lyra migrate` as a one-off deploy job before enabling new application versions.

## Backup and restore

See [DEVELOPMENT.md](DEVELOPMENT.md#backups). Restore PostgreSQL before starting workers; configure a provider-level backup/replication policy for the object bucket too.
