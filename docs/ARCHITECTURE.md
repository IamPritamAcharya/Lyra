# Architecture

Lyra v1 is one Go modular-monolith binary. `serve`, `worker`, `migrate`, `ingest`, `identify`, `eval`, and `benchmark` are execution modes, not services. PostgreSQL holds catalog and fingerprint truth; Valkey carries Asynq work; private S3-compatible storage holds reference audio. Query recordings use controlled temporary files and are deleted after synchronous identification.

Dependencies point inward: API handlers call application use cases; catalog, ingest, and identify depend on small consumer-owned interfaces; audio and fingerprint packages are deterministic and infrastructure-free. PostgreSQL will initially implement the fingerprint inverted index behind `FingerprintIndex`.

Track lifecycle is `CREATED -> UPLOADED -> INDEXING -> READY`, with `FAILED`, `REINDEXING`, and `DELETING -> DELETED` explicit state transitions. Index replacement is transactional: generate outside a transaction, then delete/version-insert/status-update in one transaction.
