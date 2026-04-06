-- 008: Production hardening
-- 1. Unique constraint on collectible identity (contract_address, chain_id, token_id)
CREATE UNIQUE INDEX IF NOT EXISTS idx_collectibles_identity
    ON collectibles (contract_address, chain_id, token_id)
    WHERE contract_address IS NOT NULL AND contract_address != '';

-- 2. Ensure notification status supports full lifecycle
-- (pending -> delivered | failed, failed -> retry -> delivered | permanently_failed)
-- The existing CHECK constraint on status is flexible enough (varchar), no ALTER needed.

-- 3. Add index for analytics event queries by type+date
CREATE INDEX IF NOT EXISTS idx_analytics_events_type_created
    ON analytics_events (event_type, created_at DESC);

-- 4. Add index for analytics events by collectible
CREATE INDEX IF NOT EXISTS idx_analytics_events_collectible
    ON analytics_events (collectible_id, event_type) WHERE collectible_id IS NOT NULL;
