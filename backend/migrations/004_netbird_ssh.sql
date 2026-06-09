-- track whether we've auto-enabled NetBird built-in SSH for this peer (once)
ALTER TABLE netbird_links ADD COLUMN IF NOT EXISTS ssh_enabled BOOLEAN NOT NULL DEFAULT FALSE;
