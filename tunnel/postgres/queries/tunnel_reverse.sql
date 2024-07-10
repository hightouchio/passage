-- name: CreateReverseTunnel :one
INSERT INTO passage.reverse_tunnels DEFAULT VALUES RETURNING *;

-- name: GetReverseTunnel :one
SELECT *
FROM passage.reverse_tunnels
WHERE id = $1;

-- name: UpdateReverseTunnel :one
UPDATE passage.reverse_tunnels
SET enabled=COALESCE(sqlc.narg(enabled), enabled)
WHERE id = $1
RETURNING *;

-- name: DeleteReverseTunnel :exec
DELETE
FROM passage.reverse_tunnels
WHERE id = $1;

-- name: ListEnabledReverseTunnels :many
SELECT rt.*, encode(sha256(array_to_string(array_agg(ka.key_id), ',')::bytea), 'hex') AS authorized_keys_hash
FROM passage.reverse_tunnels rt
         LEFT JOIN passage.key_authorizations ka ON ka.tunnel_id = rt.id
WHERE rt.enabled = true
GROUP BY rt.id;