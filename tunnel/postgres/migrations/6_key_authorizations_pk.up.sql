BEGIN;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM information_schema.constraint_table_usage
    WHERE table_schema = 'passage'
    AND table_name = 'key_authorizations'
    AND constraint_name = 'key_authorizations_pkey'
  ) THEN
    ALTER TABLE passage.key_authorizations ADD PRIMARY KEY(key_id, tunnel_type, tunnel_id);
  END IF;
END $$;

ALTER TABLE passage.key_authorizations DROP CONSTRAINT IF EXISTS key_authorizations_key_id_tunnel_type_tunnel_id_key;

COMMIT;