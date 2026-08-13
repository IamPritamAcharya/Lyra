# Architecture

Lyra v1 is one Go modular-monolith binary. `serve`, `worker`, `migrate`, `ingest`, `identify`, `eval`, and `benchmark` are execution modes, not services. PostgreSQL holds catalog and fingerprint truth; Valkey carries Asynq work; private S3-compatible storage holds reference audio. Query recordings use controlled temporary files and are deleted after synchronous identification.

The optional `web/` React/Vite static frontend is not another backend. It calls the Go API directly; browser code never contains an admin secret. Its CORS origin is explicitly restricted by `LYRA_ALLOWED_ORIGIN`.

There is one configured administrator, not a user account system. The administrator password is supplied as a bcrypt hash. Login creates a random opaque server-side session whose token hash and CSRF-token hash are stored in PostgreSQL. The browser receives only an HttpOnly cookie plus a CSRF token held in memory; every catalog write verifies both. Sessions expire after 12 hours. Local HTTP development may set `LYRA_ADMIN_COOKIE_SECURE=false`; deployed HTTPS environments must set it true.

Dependencies point inward: API handlers call application use cases; catalog, ingest, and identify depend on small consumer-owned interfaces; audio and fingerprint packages are deterministic and infrastructure-free. PostgreSQL will initially implement the fingerprint inverted index behind `FingerprintIndex`.

Track lifecycle is `CREATED -> UPLOADED -> INDEXING -> READY`, with `FAILED`, `REINDEXING`, and `DELETING -> DELETED` explicit state transitions. Index replacement is transactional: generate outside a transaction, then delete/version-insert/status-update in one transaction.

A `CREATED` track may also transition to `DELETING` so an abandoned metadata record can be removed before reference audio exists.
