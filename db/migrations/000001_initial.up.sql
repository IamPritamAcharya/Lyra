CREATE TABLE tracks (
 id BIGSERIAL PRIMARY KEY, public_id UUID UNIQUE NOT NULL, title TEXT NOT NULL, artist_name TEXT NOT NULL,
 album_name TEXT NULL, isrc TEXT NULL, musicbrainz_id UUID NULL, duration_ms INTEGER NULL,
 status TEXT NOT NULL CHECK (status IN ('CREATED','UPLOADED','INDEXING','READY','FAILED','REINDEXING','DELETING','DELETED')),
 failure_reason TEXT NULL, fingerprint_version SMALLINT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ NULL
);
CREATE TABLE track_audio (
 track_id BIGINT PRIMARY KEY REFERENCES tracks(id), object_key TEXT NOT NULL UNIQUE, sha256 TEXT NOT NULL, size_bytes BIGINT NOT NULL CHECK (size_bytes > 0), mime_type TEXT NOT NULL, original_filename TEXT NOT NULL, sample_rate INTEGER NULL, channels SMALLINT NULL, duration_ms INTEGER NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX track_audio_sha256_idx ON track_audio(sha256);
CREATE TABLE fingerprints (algorithm_version SMALLINT NOT NULL, hash INTEGER NOT NULL, track_id BIGINT NOT NULL REFERENCES tracks(id) ON DELETE CASCADE, anchor_frame INTEGER NOT NULL, PRIMARY KEY (algorithm_version, hash, track_id, anchor_frame));
CREATE INDEX fingerprints_lookup_idx ON fingerprints (algorithm_version, hash) INCLUDE (track_id, anchor_frame);
CREATE TABLE fingerprint_hash_stats (algorithm_version SMALLINT NOT NULL, hash INTEGER NOT NULL, posting_count BIGINT NOT NULL, updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), PRIMARY KEY (algorithm_version, hash));
