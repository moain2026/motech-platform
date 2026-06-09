-- track whether the agent has actually applied --allow-server-ssh (re-up done)
ALTER TABLE netbird_links ADD COLUMN IF NOT EXISTS ssh_applied BOOLEAN NOT NULL DEFAULT FALSE;
