-- 005: traffic limit per peer
-- traffic_limit_bytes = 0 means no limit
-- traffic_limited_at  = timestamp when the peer was auto-disabled due to limit

ALTER TABLE peers
  ADD COLUMN IF NOT EXISTS traffic_limit_bytes BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS traffic_limited_at  TIMESTAMPTZ;

-- index for the enforcer query: only peers with a limit set
CREATE INDEX IF NOT EXISTS idx_peers_traffic_limit
  ON peers (traffic_limit_bytes)
  WHERE traffic_limit_bytes > 0;
