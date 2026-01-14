package background

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/teamPaprika/bingsan/internal/db"
	"github.com/teamPaprika/bingsan/internal/leader"
	"github.com/teamPaprika/bingsan/internal/metrics"
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
	elector *leader.Elector
	db      *db.DB
	tasks   []Task
	wg      sync.WaitGroup
	cancel  context.CancelFunc
}

// NewRunner creates a new background task runner.
func NewRunner(elector *leader.Elector, database *db.DB) *Runner {
	return &Runner{
		elector: elector,
		db:      database,
		tasks:   DefaultTasks,
	}
}

// Start begins running all background tasks.
func (r *Runner) Start(ctx context.Context) {
	ctx, r.cancel = context.WithCancel(ctx)

	for _, task := range r.tasks {
		r.wg.Add(1)
		go r.runTask(ctx, task)
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
