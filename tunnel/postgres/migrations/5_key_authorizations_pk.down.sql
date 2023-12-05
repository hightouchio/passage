BEGIN;

ALTER TABLE passage.key_authorizations ADD CONSTRAINT key_authorizations_key_id_tunnel_type_tunnel_id_key UNIQUE (key_id, tunnel_type, tunnel_id);
ALTER TABLE passage.key_authorizations DROP CONSTRAINT IF EXISTS key_authorizations_pkey;

COMMIT;
