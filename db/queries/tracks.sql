-- name: CreateTrack :one
INSERT INTO tracks (public_id,title,artist_name,album_name,status) VALUES ($1,$2,$3,$4,'CREATED') RETURNING *;
-- name: GetTrackByPublicID :one
SELECT * FROM tracks WHERE public_id=$1 AND deleted_at IS NULL;
-- name: ListTracks :many
SELECT * FROM tracks WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2;
-- name: UpdateTrackStatus :one
UPDATE tracks SET status=$2,failure_reason=$3,updated_at=now(),deleted_at=CASE WHEN $2='DELETED' THEN now() ELSE deleted_at END WHERE public_id=$1 RETURNING *;
