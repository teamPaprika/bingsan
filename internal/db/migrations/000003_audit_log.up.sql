-- Audit logging for security and compliance
-- Records all significant catalog operations

CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    event_type TEXT NOT NULL,
    event_id TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    namespace TEXT,
    table_name TEXT,
    view_name TEXT,
    actor TEXT NOT NULL,         -- Principal ID
    actor_type TEXT NOT NULL,    -- 'api_key', 'oauth2', 'anonymous'
    ip_address INET,
    user_agent TEXT,
    request_id TEXT,
    response_code INTEGER,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp DESC);
CREATE INDEX idx_audit_log_actor ON audit_log(actor, timestamp DESC);
CREATE INDEX idx_audit_log_namespace ON audit_log(namespace, timestamp DESC);
CREATE INDEX idx_audit_log_event_type ON audit_log(event_type, timestamp DESC);
CREATE INDEX idx_audit_log_request_id ON audit_log(request_id);

-- Partial index for recent logs (most commonly queried)
CREATE INDEX idx_audit_log_recent ON audit_log(timestamp DESC)
    WHERE timestamp > NOW() - INTERVAL '7 days';

-- Function to clean up old audit logs
CREATE OR REPLACE FUNCTION cleanup_old_audit_logs(retention_days INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM audit_log
    WHERE timestamp < NOW() - (retention_days || ' days')::INTERVAL;
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;
