CREATE TABLE IF NOT EXISTS liquidity_pools (
    pool_address TEXT PRIMARY KEY,
    token_a TEXT NOT NULL,
    token_b TEXT NOT NULL,
    fee_bps INTEGER, -- basis points...something like 30 for 0.3%
    type TEXT, 
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_liquidity_pools_token_a ON liquidity_pools(token_a);
CREATE INDEX idx_liquidity_pools_token_b ON liquidity_pools(token_b);