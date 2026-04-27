-- Phase 5: multi-protocol поддержка (wireguard | amneziawg | xray).

-- Серверы получают тип транспорта и protocol-specific конфиг (Reality keys, dest, ...).
ALTER TABLE servers ADD COLUMN IF NOT EXISTS protocol TEXT NOT NULL DEFAULT 'wireguard'
    CHECK (protocol IN ('wireguard','amneziawg','xray'));
ALTER TABLE servers ADD COLUMN IF NOT EXISTS protocol_config JSONB NOT NULL DEFAULT '{}'::jsonb;

-- Peer наследует протокол от сервера. Для xray дополнительно UUID/flow/short_id.
ALTER TABLE peers ADD COLUMN IF NOT EXISTS protocol      TEXT NOT NULL DEFAULT 'wireguard';
ALTER TABLE peers ADD COLUMN IF NOT EXISTS xray_uuid     UUID;
ALTER TABLE peers ADD COLUMN IF NOT EXISTS xray_flow     TEXT NOT NULL DEFAULT '';
ALTER TABLE peers ADD COLUMN IF NOT EXISTS xray_short_id TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS peers_xray_uuid_uniq ON peers(xray_uuid) WHERE xray_uuid IS NOT NULL;
CREATE INDEX IF NOT EXISTS servers_protocol_idx ON servers(protocol);
