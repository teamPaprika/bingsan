-- Extend scan_plans table for full scan planning support

-- Add result storage column
ALTER TABLE scan_plans ADD COLUMN IF NOT EXISTS result_data JSONB;

-- Add execution metrics
ALTER TABLE scan_plans ADD COLUMN IF NOT EXISTS file_count INTEGER DEFAULT 0;
ALTER TABLE scan_plans ADD COLUMN IF NOT EXISTS files_pruned INTEGER DEFAULT 0;
ALTER TABLE scan_plans ADD COLUMN IF NOT EXISTS manifests_read INTEGER DEFAULT 0;

-- Add timing columns
ALTER TABLE scan_plans ADD COLUMN IF NOT EXISTS started_at TIMESTAMPTZ;
ALTER TABLE scan_plans ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;

-- Add error tracking
ALTER TABLE scan_plans ADD COLUMN IF NOT EXISTS error_message TEXT;

-- Improve index for background worker (poll pending plans efficiently)
DROP INDEX IF EXISTS idx_scan_plans_status;
CREATE INDEX idx_scan_plans_pending_queue
    ON scan_plans(created_at ASC)
    WHERE status = 'pending';

-- Index for checking running plans per table (prevent duplicates)
CREATE INDEX IF NOT EXISTS idx_scan_plans_running
    ON scan_plans(table_id)
    WHERE status = 'running';

-- Index for result retrieval by plan ID and status
CREATE INDEX IF NOT EXISTS idx_scan_plans_completed
    ON scan_plans(id)
    WHERE status = 'completed';

-- Add comment for documentation
COMMENT ON COLUMN scan_plans.result_data IS
    'JSONB containing file-scan-tasks, delete-files, and plan-tasks arrays';
COMMENT ON COLUMN scan_plans.file_count IS
    'Number of data files included in the scan plan';
COMMENT ON COLUMN scan_plans.files_pruned IS
    'Number of files excluded by filter predicate pushdown';
