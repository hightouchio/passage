-- name: GetNormalTunnelPrivateKeys :many
SELECT key_id
FROM passage.key_authorizations
WHERE tunnel_id = $1
  AND tunnel_type = 'normal';

-- name: GetReverseTunnelAuthorizedKeys :many
SELECT key_id
FROM passage.key_authorizations
WHERE tunnel_id = $1
  AND tunnel_type = 'normal';

-- name: AuthorizeKeyForTunnel :exec
SELECT key_id
FROM passage.key_authorizations
WHERE tunnel_id = $1
  AND tunnel_type = 'normal';
