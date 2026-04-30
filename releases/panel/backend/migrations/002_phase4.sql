-- Phase 4: client-side keygen, AmneziaWG params, mTLS, audit log.
-- Идемпотентно: миграция применяется один раз при первом старте postgres.

-- 1) peers: убираем серверное хранение приватных ключей и PSK.
ALTER TABLE peers DROP COLUMN IF EXISTS private_key_enc;
ALTER TABLE peers DROP COLUMN IF EXISTS preshared_key;

-- 2) servers: AWG-параметры (Jc/Jmin/Jmax/S1/S2/H1-H4) + альтернативные транспорты + mTLS.
ALTER TABLE servers ADD COLUMN IF NOT EXISTS awg_params JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE servers ADD COLUMN IF NOT EXISTS tcp_port  INTEGER NOT NULL DEFAULT 0;
ALTER TABLE servers ADD COLUMN IF NOT EXISTS tls_port  INTEGER NOT NULL DEFAULT 0;
ALTER TABLE servers ADD COLUMN IF NOT EXISTS agent_cert_fingerprint TEXT NOT NULL DEFAULT '';

-- 3) invites: одноразовые токены для client-side keygen flow.
CREATE TABLE IF NOT EXISTS invites (
    id            UUID PRIMARY KEY,
    token         TEXT NOT NULL UNIQUE,
    server_id     UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    suggested_name TEXT NOT NULL DEFAULT '',
    expires_at    TIMESTAMPTZ NOT NULL,
    used_at       TIMESTAMPTZ,
    peer_id       UUID REFERENCES peers(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS invites_token_idx ON invites(token);
CREATE INDEX IF NOT EXISTS invites_user_idx  ON invites(user_id);

-- 4) audit_log: журнал чувствительных действий.
CREATE TABLE IF NOT EXISTS audit_log (
    id          BIGSERIAL PRIMARY KEY,
    ts          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor_id    UUID,
    actor_email TEXT NOT NULL DEFAULT '',
    action      TEXT NOT NULL,
    target_type TEXT NOT NULL DEFAULT '',
    target_id   TEXT NOT NULL DEFAULT '',
    ip          TEXT NOT NULL DEFAULT '',
    user_agent  TEXT NOT NULL DEFAULT '',
    result      TEXT NOT NULL DEFAULT 'ok',
    meta        JSONB NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS audit_ts_idx     ON audit_log(ts DESC);
CREATE INDEX IF NOT EXISTS audit_actor_idx  ON audit_log(actor_id);
CREATE INDEX IF NOT EXISTS audit_action_idx ON audit_log(action);
