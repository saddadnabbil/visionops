ALTER TABLE cameras ADD COLUMN IF NOT EXISTS last_heartbeat_at timestamptz;
