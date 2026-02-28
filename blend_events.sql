CREATE TABLE IF NOT EXISTS blend_events (
    id BIGSERIAL PRIMARY KEY,
    ts TIMESTAMPTZ NOT NULL,
    ledger_seq INTEGER NOT NULL,
    tx_hash TEXT NOT NULL,
    contract_id TEXT NOT NULL,
    event_type TEXT NOT NULL, -- deposit, withdraw, borrow, repay, liquidate
    user_address TEXT NOT NULL,
    asset_id TEXT,
    amount NUMERIC(28, 12),
    
    -- Liquidation specific
    liquidator_address TEXT,
    collateral_asset TEXT,
    debt_asset TEXT,
    
    ingested_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_blend_events_user ON blend_events(user_address);
CREATE INDEX idx_blend_events_contract ON blend_events(contract_id);
CREATE INDEX idx_blend_events_type ON blend_events(event_type);
CREATE INDEX idx_blend_events_ts ON blend_events(ts DESC);