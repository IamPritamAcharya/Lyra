#!/usr/bin/env sh
set -eu
: "${DATABASE_URL:?DATABASE_URL must be set}"
: "${LYRA_BACKUP_DIR:?LYRA_BACKUP_DIR must be set}"
mkdir -p "$LYRA_BACKUP_DIR"
stamp=$(date -u +%Y%m%dT%H%M%SZ)
pg_dump --format=custom --file="$LYRA_BACKUP_DIR/lyra-$stamp.dump" "$DATABASE_URL"
echo "database backup written to $LYRA_BACKUP_DIR/lyra-$stamp.dump"
