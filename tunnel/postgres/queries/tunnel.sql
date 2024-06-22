-- name: GetTunnel :one
SELECT * FROM passage.reverse_tunnels WHERE id = $1;