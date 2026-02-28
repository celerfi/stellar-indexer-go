CREATE TABLE IF NOT EXISTS token_info (
    contract_address TEXT PRIMARY KEY,
    symbol TEXT NOT NULL,
    name TEXT NOT NULL,
    decimals INTEGER NOT NULL,
    total_supply TEXT NOT NULL,
    admin_address TEXT,
    is_auth_revocable BOOLEAN,
    is_mintable BOOLEAN,
    is_sac BOOLEAN NOT NULL,
    num_accounts INTEGER,
    supply_breakdown JSONB
);

CREATE INDEX idx_token_symbol ON token_info(symbol);
CREATE INDEX idx_token_name ON token_info(name);
CREATE INDEX idx_token_is_sac ON token_info(is_sac);