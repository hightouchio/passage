BEGIN;

ALTER TABLE passage.tunnels DROP COLUMN https_proxy;
ALTER TABLE passage.reverse_tunnels DROP COLUMN https_proxy;

COMMIT;