CREATE TABLE IF NOT EXISTS transaction_models (
    id BIGSERIAL PRIMARY KEY,
    block_time TIMESTAMPTZ NOT NULL,
    ledger_sequence INTEGER NOT NULL,
    transaction_hash TEXT NOT NULL,
    operation_index INTEGER NOT NULL,
    dex_name TEXT,
    source_account TEXT NOT NULL,
    token_in TEXT,
    token_out TEXT,
    offer_id BIGINT,
    dex_type TEXT,
    pool_address TEXT,
    matched_offer_id BIGINT, -- Nullable as it is optional
    buyer_account TEXT,
    seller_account TEXT,
    
    -- Using NUMERIC instead of DOUBLE PRECISION for financial accuracy
    offer_buy_amount NUMERIC,
    offer_sell_amount NUMERIC,
    amount_bought NUMERIC,
    amount_sold NUMERIC,
    offer_price NUMERIC,
    dex_fee NUMERIC,
    
    status TEXT,
    
    -- Storing the slice of structs as a JSON array
    order_matches JSONB
);

-- Recommended Indexes for performance
CREATE INDEX idx_tx_hash ON transaction_models(transaction_hash);
CREATE INDEX idx_ledger_seq ON transaction_models(ledger_sequence);
CREATE INDEX idx_source_account ON transaction_models(source_account);
CREATE INDEX idx_pool_address ON transaction_models(pool_address);