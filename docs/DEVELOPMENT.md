# Development and operations

## Backups

PostgreSQL stores Lyra's catalog and fingerprint source of truth. Create a custom-format backup with:

```bash
export DATABASE_URL='postgres://…'
export LYRA_BACKUP_DIR="$PWD/backups"
./scripts/backup.sh
```

Restore into a prepared database with `pg_restore --clean --if-exists --dbname "$DATABASE_URL" path/to/lyra-*.dump`. Back up the private S3/MinIO reference-audio bucket under a separate object-storage policy.
