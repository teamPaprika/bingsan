package background

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/kimuyb/bingsan/internal/db"
	"github.com/kimuyb/bingsan/internal/leader"
	"github.com/kimuyb/bingsan/internal/metrics"
	"github.com/kimuyb/bingsan/internal/scan"
)

// Task represents a background task that runs periodically.
type Task struct {
	Name     string
	LockKey  int64
	Interval time.Duration
	Run      func(ctx context.Context, database *db.DB) error
}

// Runner manages background task execution with leader election.
type Runner struct {
	elector     *leader.Elector
	db          *db.DB
	scanPlanner *scan.Planner
	tasks       []Task
	wg          sync.WaitGroup
	cancel      context.CancelFunc
}

// NewRunner creates a new background task runner.
func NewRunner(elector *leader.Elector, database *db.DB) *Runner {
	return &Runner{
		elector: elector,
		db:      database,
		tasks:   DefaultTasks,
	}
}

// SetScanPlanner sets the scan planner for async scan plan processing.
func (r *Runner) SetScanPlanner(planner *scan.Planner) {
	r.scanPlanner = planner
}

// Start begins running all background tasks.
func (r *Runner) Start(ctx context.Context) {
	ctx, r.cancel = context.WithCancel(ctx)

	for _, task := range r.tasks {
		r.wg.Add(1)
		go r.runTask(ctx, task)
	}

	// Start scan plan processor if planner is configured
	if r.scanPlanner != nil {
		r.wg.Add(1)
		go r.runScanPlanProcessor(ctx)
	}

	slog.Info("background task runner started", "task_count", len(r.tasks))
}

// Stop gracefully stops all background tasks.
func (r *Runner) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	r.wg.Wait()
	slog.Info("background task runner stopped")
}

// runTask runs a single task in a loop with leader election.
func (r *Runner) runTask(ctx context.Context, task Task) {
	defer r.wg.Done()

	ticker := time.NewTicker(task.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.executeTask(ctx, task)
		}
	}
}

// executeTask executes a task if this node is the leader.
func (r *Runner) executeTask(ctx context.Context, task Task) {
	// Try to acquire leadership
	isLeader, err := r.elector.TryAcquire(ctx, task.LockKey)
	if err != nil {
		slog.Error("failed to check leadership", "task", task.Name, "error", err)
		return
	}

	// Update leader status metric
	if isLeader {
		metrics.SetLeaderStatus(task.Name, 1)
	} else {
		metrics.SetLeaderStatus(task.Name, 0)
		return // Not the leader, skip execution
	}

	// Execute the task
	start := time.Now()
	err = task.Run(ctx, r.db)
	duration := time.Since(start)

	if err != nil {
		slog.Error("background task failed",
			"task", task.Name,
			"duration", duration,
			"error", err,
		)
		metrics.IncBackgroundTaskRuns(task.Name, "error")
	} else {
		slog.Debug("background task completed",
			"task", task.Name,
			"duration", duration,
		)
		metrics.IncBackgroundTaskRuns(task.Name, "success")
	}
}

// DefaultTasks contains the default background tasks.
var DefaultTasks = []Task{
	{
		Name:     "cleanup_expired_locks",
		LockKey:  leader.LockKeyCleanup,
		Interval: 1 * time.Minute,
		Run:      cleanupExpiredLocks,
	},
	{
		Name:     "cleanup_expired_tokens",
		LockKey:  leader.LockKeyCleanup,
		Interval: 5 * time.Minute,
		Run:      cleanupExpiredTokens,
	},
	{
		Name:     "cleanup_expired_scan_plans",
		LockKey:  leader.LockKeyCleanup,
		Interval: 1 * time.Minute,
		Run:      cleanupExpiredScanPlans,
	},
}

// cleanupExpiredLocks removes expired catalog locks.
func cleanupExpiredLocks(ctx context.Context, database *db.DB) error {
	result, err := database.Pool.Exec(ctx,
		"DELETE FROM catalog_locks WHERE expires_at < NOW()")
	if err != nil {
		return err
	}

	if result.RowsAffected() > 0 {
		slog.Info("cleaned up expired locks", "count", result.RowsAffected())
	}
	return nil
}

// cleanupExpiredTokens removes expired OAuth tokens and API keys.
func cleanupExpiredTokens(ctx context.Context, database *db.DB) error {
	// Clean up OAuth tokens
	result1, err := database.Pool.Exec(ctx,
		"DELETE FROM oauth_tokens WHERE expires_at < NOW()")
	if err != nil {
		return err
	}

	// Clean up expired API keys
	result2, err := database.Pool.Exec(ctx,
		"DELETE FROM api_keys WHERE expires_at IS NOT NULL AND expires_at < NOW()")
	if err != nil {
		return err
	}

	total := result1.RowsAffected() + result2.RowsAffected()
	if total > 0 {
		slog.Info("cleaned up expired tokens",
			"oauth_tokens", result1.RowsAffected(),
			"api_keys", result2.RowsAffected(),
		)
	}
	return nil
}

// cleanupExpiredScanPlans removes expired scan plans.
func cleanupExpiredScanPlans(ctx context.Context, database *db.DB) error {
	result, err := database.Pool.Exec(ctx,
		"DELETE FROM scan_plans WHERE expires_at < NOW()")
	if err != nil {
		return err
	}

	if result.RowsAffected() > 0 {
		slog.Info("cleaned up expired scan plans", "count", result.RowsAffected())
	}
	return nil
}

// runScanPlanProcessor processes pending scan plans asynchronously.
func (r *Runner) runScanPlanProcessor(ctx context.Context) {
	defer r.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.processPendingScanPlans(ctx)
		}
	}
}

// processPendingScanPlans processes a batch of pending scan plans.
func (r *Runner) processPendingScanPlans(ctx context.Context) {
	// Try to acquire leadership for scan planning
	isLeader, err := r.elector.TryAcquire(ctx, leader.LockKeyScanPlanning)
	if err != nil {
		slog.Error("failed to check scan planning leadership", "error", err)
		return
	}

	if isLeader {
		metrics.SetLeaderStatus("process_scan_plans", 1)
	} else {
		metrics.SetLeaderStatus("process_scan_plans", 0)
		return
	}

	// Process pending plans one at a time
	for {
		processed, err := r.processOneScanPlan(ctx)
		if err != nil {
			slog.Error("failed to process scan plan", "error", err)
			metrics.IncBackgroundTaskRuns("process_scan_plans", "error")
			return
		}
		if !processed {
			// No more pending plans
			return
		}
		metrics.IncBackgroundTaskRuns("process_scan_plans", "success")
	}
}

// processOneScanPlan claims and processes a single pending scan plan.
// Returns true if a plan was processed, false if no pending plans exist.
func (r *Runner) processOneScanPlan(ctx context.Context) (bool, error) {
	// Start a transaction
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	// Claim a pending plan using SKIP LOCKED to avoid contention
	var planID, tableID string

	err = tx.QueryRow(ctx, `
		SELECT id, table_id
		FROM scan_plans
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`).Scan(&planID, &tableID)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return false, nil // No pending plans
		}
		return false, err
	}

	// Mark as running
	_, err = tx.Exec(ctx, `
		UPDATE scan_plans
		SET status = 'running', started_at = NOW()
		WHERE id = $1
	`, planID)
	if err != nil {
		return false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	// Process the plan (CompletePlan reads filter/select from DB)
	slog.Info("processing scan plan", "plan_id", planID, "table_id", tableID)
	start := time.Now()

	err = r.scanPlanner.CompletePlan(ctx, planID)

	duration := time.Since(start)

	if err != nil {
		// Mark as failed
		_, updateErr := r.db.Pool.Exec(ctx, `
			UPDATE scan_plans
			SET status = 'failed', completed_at = NOW(), error_message = $2
			WHERE id = $1
		`, planID, err.Error())
		if updateErr != nil {
			slog.Error("failed to update scan plan status", "plan_id", planID, "error", updateErr)
		}
		slog.Error("scan plan processing failed",
			"plan_id", planID,
			"duration", duration,
			"error", err,
		)
		return true, nil // Return true because we did process a plan (even if it failed)
	}

	slog.Info("scan plan completed",
		"plan_id", planID,
		"duration", duration,
	)
	return true, nil
}
