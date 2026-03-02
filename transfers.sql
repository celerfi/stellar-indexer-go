CREATE TABLE IF NOT EXISTS transfers (
    id BIGSERIAL PRIMARY KEY,
    ts TIMESTAMPTZ NOT NULL,
    ledger_seq INTEGER NOT NULL,
    tx_hash TEXT NOT NULL,
    operation_index INTEGER NOT NULL,
    from_address TEXT NOT NULL,
    to_address TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    amount NUMERIC(28, 12) NOT NULL,
    ingested_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_transfers_from ON transfers(from_address);
CREATE INDEX idx_transfers_to ON transfers(to_address);
CREATE INDEX idx_transfers_asset ON transfers(asset_id);
CREATE INDEX idx_transfers_ts ON transfers(ts DESC);