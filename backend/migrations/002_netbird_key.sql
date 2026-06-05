-- 002_netbird_key.sql — store the full NetBird setup key (UUID) separately
-- from the management id. The agent needs the full key to join the mesh.
ALTER TABLE netbird_links ADD COLUMN IF NOT EXISTS setup_key_full TEXT;
