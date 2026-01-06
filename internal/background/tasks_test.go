package background

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kimuyb/bingsan/internal/db"
	"github.com/kimuyb/bingsan/internal/leader"
)

func TestTask(t *testing.T) {
	task := Task{
		Name:     "test-task",
		LockKey:  leader.LockKeyCleanup,
		Interval: time.Minute,
		Run: func(ctx context.Context, database *db.DB) error {
			return nil
		},
	}

	assert.Equal(t, "test-task", task.Name)
	assert.Equal(t, leader.LockKeyCleanup, task.LockKey)
	assert.Equal(t, time.Minute, task.Interval)
	assert.NotNil(t, task.Run)
}

func TestDefaultTasks(t *testing.T) {
	t.Run("has expected number of tasks", func(t *testing.T) {
		assert.Len(t, DefaultTasks, 3)
	})

	t.Run("all tasks have names", func(t *testing.T) {
		for _, task := range DefaultTasks {
			assert.NotEmpty(t, task.Name, "task should have a name")
		}
	})

	t.Run("all tasks have intervals", func(t *testing.T) {
		for _, task := range DefaultTasks {
			assert.Greater(t, task.Interval, time.Duration(0), "task %s should have positive interval", task.Name)
		}
	})

	t.Run("all tasks have run functions", func(t *testing.T) {
		for _, task := range DefaultTasks {
			assert.NotNil(t, task.Run, "task %s should have a run function", task.Name)
		}
	})

	t.Run("cleanup_expired_locks task exists", func(t *testing.T) {
		found := false
		for _, task := range DefaultTasks {
			if task.Name == "cleanup_expired_locks" {
				found = true
				assert.Equal(t, 1*time.Minute, task.Interval)
				break
			}
		}
		assert.True(t, found, "cleanup_expired_locks task should exist")
	})

	t.Run("cleanup_expired_tokens task exists", func(t *testing.T) {
		found := false
		for _, task := range DefaultTasks {
			if task.Name == "cleanup_expired_tokens" {
				found = true
				assert.Equal(t, 5*time.Minute, task.Interval)
				break
			}
		}
		assert.True(t, found, "cleanup_expired_tokens task should exist")
	})

	t.Run("cleanup_expired_scan_plans task exists", func(t *testing.T) {
		found := false
		for _, task := range DefaultTasks {
			if task.Name == "cleanup_expired_scan_plans" {
				found = true
				assert.Equal(t, 1*time.Minute, task.Interval)
				break
			}
		}
		assert.True(t, found, "cleanup_expired_scan_plans task should exist")
	})
}

func TestNewRunner(t *testing.T) {
	elector := leader.NewElector(nil, "test-node")
	runner := NewRunner(elector, nil)

	require.NotNil(t, runner)
	assert.Same(t, elector, runner.elector)
	assert.Nil(t, runner.db)
	assert.Len(t, runner.tasks, len(DefaultTasks))
}

func TestRunner_Stop_NotStarted(t *testing.T) {
	elector := leader.NewElector(nil, "test-node")
	runner := NewRunner(elector, nil)

	// Should not panic when stopping a runner that was never started
	runner.Stop()
}

func TestRunner_StartStop(t *testing.T) {
	elector := leader.NewElector(nil, "test-node")
	runner := &Runner{
		elector: elector,
		db:      nil,
		tasks:   []Task{}, // Empty tasks to avoid DB operations
	}

	ctx := context.Background()
	runner.Start(ctx)

	// Give a moment for goroutines to start
	time.Sleep(10 * time.Millisecond)

	// Stop should not hang
	done := make(chan struct{})
	go func() {
		runner.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Fatal("Stop() timed out")
	}
}
