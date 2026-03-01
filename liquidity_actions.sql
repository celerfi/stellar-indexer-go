CREATE TABLE IF NOT EXISTS liquidity_actions (
    id BIGSERIAL PRIMARY KEY,
    ts TIMESTAMPTZ NOT NULL,
    ledger_seq INTEGER NOT NULL,
    tx_hash TEXT NOT NULL,
    pool_address TEXT NOT NULL REFERENCES liquidity_pools(pool_address),
    action_type TEXT NOT NULL, -- deposit, withdraw
    user_address TEXT NOT NULL,
    amount_a NUMERIC(28, 12) NOT NULL,
    amount_b NUMERIC(28, 12) NOT NULL,
    token_a TEXT NOT NULL,
    token_b TEXT NOT NULL,
    ingested_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_liq_actions_pool ON liquidity_actions(pool_address);
CREATE INDEX idx_liq_actions_user ON liquidity_actions(user_address);
CREATE INDEX idx_liq_actions_ts ON liquidity_actions(ts DESC);