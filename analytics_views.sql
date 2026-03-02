-- Current reserves for each pool based on net liquidity actions
CREATE MATERIALIZED VIEW IF NOT EXISTS pool_reserves AS
SELECT 
    pool_address,
    token_a,
    token_b,
    SUM(CASE WHEN action_type = 'deposit' THEN amount_a ELSE -amount_a END) as reserve_a,
    SUM(CASE WHEN action_type = 'deposit' THEN amount_b ELSE -amount_b END) as reserve_b,
    MAX(ts) as last_update
FROM liquidity_actions
GROUP BY pool_address, token_a, token_b
WITH NO DATA;

-- Real-time TVL per pool calculated by multiplying reserves by latest USD price
CREATE MATERIALIZED VIEW IF NOT EXISTS pool_tvl AS
WITH latest_prices AS (
    SELECT DISTINCT ON (asset_id) asset_id, price_usd
    FROM price_ticks
    ORDER BY asset_id, ts DESC
)
SELECT 
    r.pool_address,
    (r.reserve_a * COALESCE(pa.price_usd, 0)) + (r.reserve_b * COALESCE(pb.price_usd, 0)) as tvl_usd,
    r.last_update
FROM pool_reserves r
LEFT JOIN latest_prices pa ON r.token_a = pa.asset_id
LEFT JOIN latest_prices pb ON r.token_b = pb.asset_id
WITH NO DATA;

-- 24-hour trading volume per pool
CREATE MATERIALIZED VIEW IF NOT EXISTS pool_volume_24h AS
SELECT 
    pool_address,
    dex_name,
    SUM(amount_bought) as volume_token_in, -- Simplified
    COUNT(*) as trade_count
FROM transaction_models
WHERE block_time > NOW() - INTERVAL '24 hours'
GROUP BY pool_address, dex_name
WITH NO DATA;

-- Refresh function to be called periodically
CREATE OR REPLACE FUNCTION refresh_analytics_views()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY pool_reserves;
    REFRESH MATERIALIZED VIEW CONCURRENTLY pool_tvl;
    REFRESH MATERIALIZED VIEW CONCURRENTLY pool_volume_24h;
END;
$$ LANGUAGE plpgsql;

-- Initial indexes for fast retrieval
CREATE INDEX IF NOT EXISTS idx_pool_tvl_val ON pool_tvl(tvl_usd DESC);
CREATE INDEX IF NOT EXISTS idx_pool_reserves_addr ON pool_reserves(pool_address);