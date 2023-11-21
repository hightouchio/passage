BEGIN;

ALTER TABLE passage.tunnels ADD COLUMN tunnel_port INT DEFAULT nextval('passage.tunnel_ports') UNIQUE;
ALTER TABLE passage.reverse_tunnels ADD COLUMN tunnel_port INT DEFAULT nextval('passage.tunnel_ports') UNIQUE;

COMMIT;
