-- track the OS login user reported by the agent (for connection info)
ALTER TABLE clients ADD COLUMN IF NOT EXISTS login_user TEXT;
