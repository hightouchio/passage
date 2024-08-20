ALTER TABLE passage.tunnels ADD COLUMN IF NOT EXISTS healthcheck_config JSONB;
ALTER TABLE passage.reverse_tunnels ADD COLUMN IF NOT EXISTS healthcheck_config JSONB;
