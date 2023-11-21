BEGIN;

ALTER TABLE passage.tunnels DROP COLUMN error;
ALTER TABLE passage.reverse_tunnels DROP COLUMN error;

COMMIT;
