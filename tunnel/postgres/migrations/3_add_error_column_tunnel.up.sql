ALTER TABLE passage.tunnels ADD COLUMN IF NOT EXISTS error text;
ALTER TABLE passage.reverse_tunnels ADD COLUMN IF NOT EXISTS error text;