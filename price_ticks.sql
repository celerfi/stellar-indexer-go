CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS assets (
    asset_id          TEXT        PRIMARY KEY,
    asset_code        TEXT        NOT NULL,
    asset_type        TEXT        NOT NULL CHECK (asset_type IN ('classic', 'soroban')),
    issuer_address    TEXT,
    contract_address  TEXT,
    home_domain       TEXT,
    asset_name        TEXT,
    decimals          SMALLINT    NOT NULL DEFAULT 7,
    is_active         BOOLEAN     NOT NULL DEFAULT TRUE,
    first_seen_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sources (
    source_id        TEXT     PRIMARY KEY,
    display_name     TEXT     NOT NULL,
    source_type      TEXT     NOT NULL CHECK (source_type IN ('dex', 'amm', 'oracle_onchain', 'oracle_offchain', 'cex')),
    reports_volume   BOOLEAN  NOT NULL DEFAULT FALSE,
    is_enabled       BOOLEAN  NOT NULL DEFAULT TRUE,
    health_status    TEXT     NOT NULL DEFAULT 'healthy' CHECK (health_status IN ('healthy', 'degraded', 'unavailable')),
    last_tick_at     TIMESTAMPTZ,
    config           JSONB
);

INSERT INTO sources (source_id, display_name, source_type, reports_volume) VALUES
    ('reflector', 'Reflector Oracle', 'oracle_onchain',  FALSE),
    ('soroswap',  'Soroswap',         'amm',             TRUE),
    ('aquarius',  'Aquarius',         'amm',             TRUE),
    ('sdex',      'Stellar DEX',      'dex',             TRUE),
    ('redstone',  'Redstone',         'oracle_offchain', FALSE),
    ('chainlink', 'Chainlink',        'oracle_offchain', FALSE),
    ('seda',      'SEDA',             'oracle_offchain', FALSE)
ON CONFLICT (source_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS price_ticks (
    ts            TIMESTAMPTZ     NOT NULL,
    asset_id      TEXT            NOT NULL REFERENCES assets(asset_id),
    source_id     TEXT            NOT NULL REFERENCES sources(source_id),
    source_type   TEXT            NOT NULL,
    price_usd     NUMERIC(28, 12) NOT NULL,
    volume_usd    NUMERIC(28, 12),
    base_volume   NUMERIC(28, 12),
    quote_volume  NUMERIC(28, 12),
    ledger_seq    BIGINT,
    tx_hash       TEXT,
    raw_payload   JSONB,
    ingested_at   TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

SELECT create_hypertable('price_ticks', 'ts', if_not_exists => TRUE);

ALTER TABLE price_ticks SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'asset_id, source_id',
    timescaledb.compress_orderby   = 'ts DESC'
);

SELECT add_compression_policy('price_ticks', INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_retention_policy('price_ticks', INTERVAL '90 days', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS price_ticks_asset_ts_idx  ON price_ticks (asset_id, ts DESC);
CREATE INDEX IF NOT EXISTS price_ticks_source_ts_idx ON price_ticks (source_id, ts DESC);

-- 1-minute per source
CREATE MATERIALIZED VIEW IF NOT EXISTS ohlcv_1min_by_source
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 minute', ts)                          AS bucket,
    asset_id,
    source_id,
    FIRST(price_usd, ts)                                 AS open,
    MAX(price_usd)                                       AS high,
    MIN(price_usd)                                       AS low,
    LAST(price_usd, ts)                                  AS close,
    SUM(price_usd * volume_usd) / NULLIF(SUM(volume_usd), 0) AS vwap,
    SUM(volume_usd)                                      AS volume_usd,
    SUM(base_volume)                                     AS base_volume,
    SUM(quote_volume)                                    AS quote_volume,
    COUNT(*)                                             AS tick_count
FROM price_ticks
GROUP BY bucket, asset_id, source_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('ohlcv_1min_by_source',
    start_offset      => INTERVAL '10 minutes',
    end_offset        => INTERVAL '30 seconds',
    schedule_interval => INTERVAL '30 seconds',
    if_not_exists     => TRUE
);

SELECT add_retention_policy('ohlcv_1min_by_source', INTERVAL '90 days', if_not_exists => TRUE);

-- 1-minute cross-source
CREATE MATERIALIZED VIEW IF NOT EXISTS ohlcv_1min
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 minute', bucket)                      AS bucket,
    asset_id,
    FIRST(open, bucket)                                  AS open,
    MAX(high)                                            AS high,
    MIN(low)                                             AS low,
    LAST(close, bucket)                                  AS close,
    SUM(vwap * volume_usd) / NULLIF(SUM(volume_usd), 0) AS vwap,
    SUM(volume_usd)                                      AS volume_usd,
    SUM(base_volume)                                     AS base_volume,
    SUM(quote_volume)                                    AS quote_volume,
    SUM(tick_count)                                      AS tick_count
FROM ohlcv_1min_by_source
GROUP BY time_bucket('1 minute', bucket), asset_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('ohlcv_1min',
    start_offset      => INTERVAL '10 minutes',
    end_offset        => INTERVAL '1 minute',
    schedule_interval => INTERVAL '1 minute',
    if_not_exists     => TRUE
);

SELECT add_retention_policy('ohlcv_1min', INTERVAL '30 days', if_not_exists => TRUE);

-- 1-hour
CREATE MATERIALIZED VIEW IF NOT EXISTS ohlcv_1hour
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', bucket)                        AS bucket,
    asset_id,
    FIRST(open, bucket)                                  AS open,
    MAX(high)                                            AS high,
    MIN(low)                                             AS low,
    LAST(close, bucket)                                  AS close,
    SUM(vwap * volume_usd) / NULLIF(SUM(volume_usd), 0) AS vwap,
    SUM(volume_usd)                                      AS volume_usd,
    SUM(base_volume)                                     AS base_volume,
    SUM(quote_volume)                                    AS quote_volume,
    SUM(tick_count)                                      AS tick_count
FROM ohlcv_1min
GROUP BY time_bucket('1 hour', bucket), asset_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('ohlcv_1hour',
    start_offset      => INTERVAL '4 hours',
    end_offset        => INTERVAL '2 hours',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists     => TRUE
);

-- 1-day
CREATE MATERIALIZED VIEW IF NOT EXISTS ohlcv_1day
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 day', bucket)                         AS bucket,
    asset_id,
    FIRST(open, bucket)                                  AS open,
    MAX(high)                                            AS high,
    MIN(low)                                             AS low,
    LAST(close, bucket)                                  AS close,
    SUM(vwap * volume_usd) / NULLIF(SUM(volume_usd), 0) AS vwap,
    SUM(volume_usd)                                      AS volume_usd,
    SUM(base_volume)                                     AS base_volume,
    SUM(quote_volume)                                    AS quote_volume,
    SUM(tick_count)                                      AS tick_count
FROM ohlcv_1hour
GROUP BY time_bucket('1 day', bucket), asset_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('ohlcv_1day',
    start_offset      => INTERVAL '4 days',
    end_offset        => INTERVAL '2 days',
    schedule_interval => INTERVAL '1 day',
    if_not_exists     => TRUE
);