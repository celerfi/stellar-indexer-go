CREATE TYPE available_chains AS ENUM ('stellar');
CREATE TYPE service_type AS ENUM ('indexer', 'rpc');

CREATE TABLE IF NOT EXISTS b2b_request_logs (
    log_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       UUID NOT NULL,
    api_key_id       UUID NOT NULL,
    chain            available_chains NOT NULL,
    service_id       VARCHAR(100) NOT NULL,
    service_type     service_type NOT NULL,
    endpoint_path    VARCHAR(255),
    http_method      VARCHAR(10),
    received_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at     TIMESTAMPTZ,
    duration_ms      BIGINT,
    status           VARCHAR(20) NOT NULL,
    http_status_code SMALLINT,
    error_code       VARCHAR(100),
    error_message    TEXT,
    credits_used     DECIMAL(18, 8) NOT NULL DEFAULT 0.0,
    source_ip        VARCHAR(45),
    location         VARCHAR(100),
    correlation_id   UUID
);

CREATE INDEX idx_logs_project_time ON b2b_request_logs (project_id, received_at DESC);
CREATE INDEX idx_logs_service_status_time ON b2b_request_logs (service_id, status, received_at DESC);
CREATE INDEX idx_logs_source_ip ON b2b_request_logs (source_ip);
CREATE INDEX idx_logs_api_key ON b2b_request_logs (api_key_id);