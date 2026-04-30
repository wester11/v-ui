-- Phase 6: server/config split + node auto-registration.

-- Expand protocol enum-like CHECK for infra-only servers.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'servers_protocol_check'
    ) THEN
        ALTER TABLE servers DROP CONSTRAINT servers_protocol_check;
    END IF;
END $$;

ALTER TABLE servers
    ADD CONSTRAINT servers_protocol_check
    CHECK (protocol IN ('none','wireguard','amneziawg','xray'));

ALTER TABLE servers ADD COLUMN IF NOT EXISTS node_id UUID;
ALTER TABLE servers ADD COLUMN IF NOT EXISTS node_secret TEXT NOT NULL DEFAULT '';
ALTER TABLE servers ADD COLUMN IF NOT EXISTS hostname TEXT NOT NULL DEFAULT '';
ALTER TABLE servers ADD COLUMN IF NOT EXISTS ip TEXT NOT NULL DEFAULT '';
ALTER TABLE servers ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE servers ADD COLUMN IF NOT EXISTS agent_version TEXT NOT NULL DEFAULT '';

UPDATE servers SET node_id = id WHERE node_id IS NULL;
ALTER TABLE servers ALTER COLUMN node_id SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS servers_node_id_uniq ON servers(node_id);
CREATE INDEX IF NOT EXISTS servers_status_idx ON servers(status);

CREATE TABLE IF NOT EXISTS vpn_configs (
    id            UUID PRIMARY KEY,
    server_id     UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    protocol      TEXT NOT NULL CHECK (protocol IN ('wireguard','amneziawg','xray')),
    template      TEXT NOT NULL DEFAULT 'vless_reality',
    setup_mode    TEXT NOT NULL DEFAULT 'simple' CHECK (setup_mode IN ('simple','advanced')),
    routing_mode  TEXT NOT NULL DEFAULT 'simple' CHECK (routing_mode IN ('simple','advanced','cascade')),
    settings      JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_active     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS vpn_configs_server_idx ON vpn_configs(server_id);
CREATE UNIQUE INDEX IF NOT EXISTS vpn_configs_active_per_server
    ON vpn_configs(server_id)
    WHERE is_active = TRUE;

