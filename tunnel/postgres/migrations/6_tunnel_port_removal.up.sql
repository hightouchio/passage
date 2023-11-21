BEGIN;

ALTER TABLE passage.tunnels DROP COLUMN tunnel_port;
ALTER TABLE passage.reverse_tunnels DROP COLUMN tunnel_port;

COMMIT;
