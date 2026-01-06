-- Revert audit logging

DROP FUNCTION IF EXISTS cleanup_old_audit_logs(INTEGER);
DROP INDEX IF EXISTS idx_audit_log_recent;
DROP INDEX IF EXISTS idx_audit_log_request_id;
DROP INDEX IF EXISTS idx_audit_log_event_type;
DROP INDEX IF EXISTS idx_audit_log_namespace;
DROP INDEX IF EXISTS idx_audit_log_actor;
DROP INDEX IF EXISTS idx_audit_log_timestamp;
DROP TABLE IF EXISTS audit_log;
