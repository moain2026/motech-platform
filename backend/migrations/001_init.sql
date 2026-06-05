-- 001_init.sql — initial schema for Motech Platform
-- Standard PostgreSQL only (portable). UUIDs via pgcrypto's gen_random_uuid().

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- admins: dashboard users (self-managed auth, JWT + bcrypt)
CREATE TABLE IF NOT EXISTS admins (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'admin',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- clients: managed remote machines (branches/companies)
CREATE TABLE IF NOT EXISTS clients (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    branch        TEXT,
    contact_name  TEXT,
    contact_phone TEXT,
    status        TEXT NOT NULL DEFAULT 'pending', -- pending|online|offline|disabled
    last_seen     TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- setup_tokens: one-time activation tokens (stored hashed)
CREATE TABLE IF NOT EXISTS setup_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id  UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    used_at    TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ssh_keys: per-client unique SSH keys (private key encrypted at rest, optional)
CREATE TABLE IF NOT EXISTS ssh_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id       UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    public_key      TEXT,
    private_key_enc BYTEA,
    fingerprint     TEXT,
    active          BOOLEAN NOT NULL DEFAULT true,
    rotated_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- netbird_links: mapping client -> NetBird peer/group/setup-key
CREATE TABLE IF NOT EXISTS netbird_links (
    client_id     UUID PRIMARY KEY REFERENCES clients(id) ON DELETE CASCADE,
    peer_id       TEXT,
    setup_key_ref TEXT,
    group_id      TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- activity_log: audit trail of all sensitive actions
CREATE TABLE IF NOT EXISTS activity_log (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor      TEXT NOT NULL,
    client_id  UUID REFERENCES clients(id) ON DELETE SET NULL,
    action     TEXT NOT NULL,
    metadata   JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_activity_created ON activity_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_clients_status ON clients(status);
