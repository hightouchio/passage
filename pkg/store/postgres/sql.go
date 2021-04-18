package postgres

const createTunnel = `
INSERT INTO tunnels (id, public_key, private_key, port, server_endpoint, server_port, service_endpoint, service_port)
VALUES ($1, $2, $3, $4, $5)
`

const getTunnel = `
SELECT id, created_at, public_key, private_key, port, server_endpoint, server_port, service_endpoint, service_port
FROM tunnels
WHERE id = $1
`

const listTunnels = `
SELECT id, created_at, public_key, private_key, port, server_endpoint, server_port, service_endpoint, service_port
FROM tunnels
`

const createReverseTunnel = `
INSERT INTO reverse_tunnels (id, public_key, private_key, port, ssh_port)
VALUES ($1, $2, $3, $4, $5, $6)
`

const getReverseTunnel = `
SELECT id, created_at, public_key, private_key, port, ssh_port
FROM reverse_tunnels
WHERE id = $1
`

const listReverseTunnels = `
SELECT id, created_at, public_key, private_key, port, ssh_port
FROM reverse_tunnels
`
