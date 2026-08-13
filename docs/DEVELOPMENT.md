# Development and operations

## Frontend

The public browser interface lives in `web/`. Its Node dependencies use Vite 5 for compatibility with Node 18; use Node 20.19+ before upgrading to newer Vite versions.

```bash
cd web
cp .env.example .env
npm install
npm run dev
```

Start the Go API separately and set `LYRA_ALLOWED_ORIGIN=http://localhost:5173`. The frontend’s `VITE_LYRA_API_BASE_URL` must point to that API.

The browser admin is a single configured account. Generate its bcrypt password hash once, put only that hash in `.env`, and keep the plain-text password out of the repository:

```bash
htpasswd -bnBC 12 "" 'choose-a-long-unique-password' | tr -d ':\n'
```

Set the result as `LYRA_ADMIN_PASSWORD_HASH`; optionally change `LYRA_ADMIN_USERNAME`. `LYRA_ADMIN_COOKIE_SECURE=false` is for local HTTP only. Set it to `true` for HTTPS deployments. The browser receives an HttpOnly session cookie and an in-memory CSRF token—not an admin secret.

## Backups

PostgreSQL stores Lyra's catalog and fingerprint source of truth. Create a custom-format backup with:

```bash
export DATABASE_URL='postgres://…'
export LYRA_BACKUP_DIR="$PWD/backups"
./scripts/backup.sh
```

Restore into a prepared database with `pg_restore --clean --if-exists --dbname "$DATABASE_URL" path/to/lyra-*.dump`. Back up the private S3/MinIO reference-audio bucket under a separate object-storage policy.
