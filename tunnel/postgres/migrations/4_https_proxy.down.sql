BEGIN;

ALTER TABLE passage.tunnels DROP COLUMN http_proxy;
ALTER TABLE passage.reverse_tunnels DROP COLUMN http_proxy;

COMMIT;