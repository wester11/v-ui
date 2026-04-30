-- void-wg schema

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY,
    email           TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    role            TEXT NOT NULL CHECK (role IN ('admin','operator','user')),
    disabled        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS servers (
    id              UUID PRIMARY KEY,
    name            TEXT NOT NULL,
    endpoint        TEXT NOT NULL,
    public_key      TEXT NOT NULL,
    listen_port     INTEGER NOT NULL,
    subnet          TEXT NOT NULL,
    dns             TEXT NOT NULL DEFAULT '',
    obfs_enabled    BOOLEAN NOT NULL DEFAULT FALSE,
    agent_token     TEXT NOT NULL UNIQUE,
    online          BOOLEAN NOT NULL DEFAULT FALSE,
    last_heartbeat  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS peers (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id       UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    public_key      TEXT NOT NULL,
    private_key_enc BYTEA NOT NULL,
    preshared_key   TEXT NOT NULL,
    assigned_ip     TEXT NOT NULL,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    bytes_rx        BIGINT NOT NULL DEFAULT 0,
    bytes_tx        BIGINT NOT NULL DEFAULT 0,
    last_handshake  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL,
    UNIQUE(server_id, assigned_ip)
);

CREATE INDEX IF NOT EXISTS peers_user_idx   ON peers(user_id);
CREATE INDEX IF NOT EXISTS peers_server_idx ON peers(server_id);

-- Bootstrap-админ создаётся приложением на старте из env
-- BOOTSTRAP_ADMIN_EMAIL / BOOTSTRAP_ADMIN_PASSWORD (см. backend/internal/bootstrap).
