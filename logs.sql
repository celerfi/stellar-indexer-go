CREATE TYPE available_chains AS ENUM ('stellar')
CREATE TYPE service_type AS ENUM ('indexer', 'rpc')

CREATE TABLE IF NOT EXISTS b2b_request_logs (
    log_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id_id UUID NOT NULL, -- tentative: project_id
    chain available_chains NOT NULL,
    service_id VARCHAR(100) NOT NULL,
    service_type service_type not null,
    endpoint_path VARCHAR(255),
    http_method VARCHAR(10),
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER,
    status VARCHAR(20) NOT NULL,
    http_status_code SMALLINT,
    error_code VARCHAR(100) NULL,
    error_message TEXT NULL,
    credits_used DECIMAL(18, 8) NOT NULL DEFAULT 0.0,
    source_ip VARCHAR(45),
    location VARCHAR(45), -- location datatype
    correlation_id UUID NULL -- maybe
);

CREATE INDEX idx_logs_project_time ON b2b_request_logs (project_id, received_at DESC);
CREATE INDEX idx_logs_service_status_time ON b2b_request_logs (service_id, status, received_at DESC);
CREATE INDEX idx_logs_source_ip ON b2b_request_logs (source_ip);
CREATE INDEX idx_logs_api_key ON b2b_request_logs (api_key_id);
-- CREATE INDEX idx_logs_correlation_id ON b2b_request_logs (correlation_id);