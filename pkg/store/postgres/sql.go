package postgres

const createTunnel = `
INSERT INTO tunnels (id, type, public_key, private_key, port)
VALUES ($1, $2, $3, $4, $5)
`

const getTunnel = `
SELECT id, created_at, type, public_key, private_key, port
FROM tunnels
WHERE id = $1
`

const listTunnels = `
SELECT id, created_at, type, public_key, private_key, port
FROM tunnels
`
