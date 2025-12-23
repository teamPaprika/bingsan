-- Drop functions
DROP FUNCTION IF EXISTS release_lock(TEXT, TEXT);
DROP FUNCTION IF EXISTS try_acquire_lock(TEXT, TEXT, INTEGER);
DROP FUNCTION IF EXISTS cleanup_expired_locks();
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;

-- Drop triggers (will be dropped with function CASCADE, but explicit for clarity)
DROP TRIGGER IF EXISTS update_namespaces_updated_at ON namespaces;
DROP TRIGGER IF EXISTS update_tables_updated_at ON tables;
DROP TRIGGER IF EXISTS update_views_updated_at ON views;

-- Drop indexes (will be dropped with tables, but explicit for clarity)
DROP INDEX IF EXISTS idx_namespaces_name;
DROP INDEX IF EXISTS idx_tables_namespace;
DROP INDEX IF EXISTS idx_tables_name;
DROP INDEX IF EXISTS idx_views_namespace;
DROP INDEX IF EXISTS idx_views_name;
DROP INDEX IF EXISTS idx_locks_expires;
DROP INDEX IF EXISTS idx_commit_log_table;
DROP INDEX IF EXISTS idx_scan_plans_table;
DROP INDEX IF EXISTS idx_scan_plans_status;

-- Drop tables in reverse order of creation (respecting foreign keys)
DROP TABLE IF EXISTS scan_plans;
DROP TABLE IF EXISTS oauth_tokens;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS commit_log;
DROP TABLE IF EXISTS catalog_locks;
DROP TABLE IF EXISTS views;
DROP TABLE IF EXISTS tables;
DROP TABLE IF EXISTS namespaces;

-- Drop extension (optional, might be used by other things)
-- DROP EXTENSION IF EXISTS "pgcrypto";
