CREATE TABLE collectibles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id UUID NOT NULL REFERENCES users(id),
    title VARCHAR(300) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    contract_address VARCHAR(42),
    chain_id INT,
    token_id VARCHAR(78),
    metadata_uri TEXT,
    image_url TEXT,
    price_cents BIGINT NOT NULL CHECK (price_cents > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    status VARCHAR(20) NOT NULL DEFAULT 'published' CHECK (status IN ('published', 'hidden')),
    hidden_by UUID REFERENCES users(id),
    hidden_reason TEXT,
    view_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_collectibles_status_created ON collectibles(status, created_at DESC);
CREATE INDEX idx_collectibles_seller ON collectibles(seller_id);

CREATE TABLE collectible_tx_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    collectible_id UUID NOT NULL REFERENCES collectibles(id) ON DELETE CASCADE,
    tx_hash VARCHAR(66) NOT NULL,
    from_address VARCHAR(42) NOT NULL,
    to_address VARCHAR(42) NOT NULL,
    block_number BIGINT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_collectible_tx_collectible ON collectible_tx_history(collectible_id, timestamp DESC);
