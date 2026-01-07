-- Rollback scan planning extensions

-- Remove indexes
DROP INDEX IF EXISTS idx_scan_plans_completed;
DROP INDEX IF EXISTS idx_scan_plans_running;
DROP INDEX IF EXISTS idx_scan_plans_pending_queue;

-- Restore original index
CREATE INDEX IF NOT EXISTS idx_scan_plans_status
    ON scan_plans(status)
    WHERE status IN ('pending', 'running');

-- Remove columns
ALTER TABLE scan_plans DROP COLUMN IF EXISTS error_message;
ALTER TABLE scan_plans DROP COLUMN IF EXISTS completed_at;
ALTER TABLE scan_plans DROP COLUMN IF EXISTS started_at;
ALTER TABLE scan_plans DROP COLUMN IF EXISTS manifests_read;
ALTER TABLE scan_plans DROP COLUMN IF EXISTS files_pruned;
ALTER TABLE scan_plans DROP COLUMN IF EXISTS file_count;
ALTER TABLE scan_plans DROP COLUMN IF EXISTS result_data;
