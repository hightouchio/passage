-- name: CreateNormalTunnel :one
INSERT INTO passage.tunnels (ssh_user, ssh_host, ssh_port, service_host, service_port)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetNormalTunnel :one
SELECT *
FROM passage.tunnels
WHERE id = $1;

-- name: UpdateNormalTunnel :one
UPDATE passage.tunnels
SET enabled=COALESCE(sqlc.narg(enabled), enabled),
    service_host=COALESCE(sqlc.narg(service_host), service_host),
    service_port=COALESCE(sqlc.narg(service_port), service_port),
    ssh_host=COALESCE(sqlc.narg(ssh_host), ssh_host),
    ssh_port=COALESCE(sqlc.narg(ssh_port), ssh_port),
    ssh_user=COALESCE(sqlc.narg(ssh_user), ssh_user)
WHERE id = $1
RETURNING *;

-- name: DeleteNormalTunnel :exec
    DELETE
FROM passage.tunnels
WHERE id = $1;

-- name: ListEnabledNormalTunnels :many
SELECT *
FROM passage.tunnels
WHERE enabled = true;
