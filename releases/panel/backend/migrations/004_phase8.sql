-- Phase 8: distributed-system kernel.
-- - jobs queue (control-plane → agent pull)
-- - config_versions (rollback / drift comparison)
-- - server: extended state model + drift hash + system metrics columns
-- Применяется автоматически EnsureSchemaUpgrades.

CREATE TABLE IF NOT EXISTS jobs (
    id           UUID PRIMARY KEY,
    server_id    UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    type         TEXT NOT NULL,                   -- deploy | restart | update | rotate-secret | exec
    status       TEXT NOT NULL DEFAULT 'pending', -- pending | running | success | failed | cancelled
    payload      JSONB NOT NULL DEFAULT '{}'::jsonb,
    result       JSONB NOT NULL DEFAULT '{}'::jsonb,
    error        TEXT NOT NULL DEFAULT '',
    attempts     INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    next_run_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at   TIMESTAMPTZ,
    finished_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS jobs_server_pending_idx
    ON jobs(server_id, status, next_run_at)
    WHERE status IN ('pending','running');
CREATE INDEX IF NOT EXISTS jobs_created_idx ON jobs(created_at DESC);

CREATE TABLE IF NOT EXISTS config_versions (
    id          UUID PRIMARY KEY,
    server_id   UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    version     INTEGER NOT NULL,
    config_json JSONB NOT NULL,
    config_hash TEXT NOT NULL,             -- sha256 hex over config_json bytes
    status      TEXT NOT NULL DEFAULT 'active',  -- active | failed | rolled_back
    note        TEXT NOT NULL DEFAULT '',
    actor_id    UUID,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(server_id, version)
);
CREATE INDEX IF NOT EXISTS config_versions_server_idx ON config_versions(server_id, version DESC);

-- Drift detection: agent reports actual config hash; control-plane знает
-- expected (last deployed). При расхождении server.status -> 'drifted'.
ALTER TABLE servers ADD COLUMN IF NOT EXISTS config_hash_expected TEXT NOT NULL DEFAULT '';
ALTER TABLE servers ADD COLUMN IF NOT EXISTS config_hash_actual   TEXT NOT NULL DEFAULT '';

-- System metrics (last reported via heartbeat).
ALTER TABLE servers ADD COLUMN IF NOT EXISTS cpu_pct      DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE servers ADD COLUMN IF NOT EXISTS ram_pct      DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE servers ADD COLUMN IF NOT EXISTS load_avg     DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE servers ADD COLUMN IF NOT EXISTS error_reason TEXT NOT NULL DEFAULT '';
