package leader

import (
	"context"
	"log/slog"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Advisory lock keys (arbitrary int64 values for PostgreSQL advisory locks)
const (
	LockKeyCleanup      int64 = 1 // For cleanup background tasks
	LockKeyMetrics      int64 = 2 // For metrics aggregation tasks
	LockKeyScanPlanning int64 = 3 // For async scan plan processing
)

// Elector manages leader election using PostgreSQL advisory locks.
type Elector struct {
	pool   *pgxpool.Pool
	nodeID string
	mu     sync.Mutex
	locks  map[int64]bool
}

// NewElector creates a new leader elector.
func NewElector(pool *pgxpool.Pool, nodeID string) *Elector {
	return &Elector{
		pool:   pool,
		nodeID: nodeID,
		locks:  make(map[int64]bool),
	}
}

// TryAcquire attempts to acquire leadership for a given lock key.
// Returns true if this node is now the leader, false otherwise.
// Uses PostgreSQL's pg_try_advisory_lock which is session-level and non-blocking.
func (e *Elector) TryAcquire(ctx context.Context, lockKey int64) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Already holding this lock
	if e.locks[lockKey] {
		return true, nil
	}

	var acquired bool
	err := e.pool.QueryRow(ctx,
		"SELECT pg_try_advisory_lock($1)", lockKey).Scan(&acquired)
	if err != nil {
		return false, err
	}

	if acquired {
		e.locks[lockKey] = true
		slog.Info("acquired leadership",
			"node_id", e.nodeID,
			"lock_key", lockKey,
		)
	}

	return acquired, nil
}

// Release releases leadership for a given lock key.
func (e *Elector) Release(ctx context.Context, lockKey int64) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.locks[lockKey] {
		return nil // Not holding this lock
	}

	_, err := e.pool.Exec(ctx, "SELECT pg_advisory_unlock($1)", lockKey)
	if err != nil {
		return err
	}

	delete(e.locks, lockKey)
	slog.Info("released leadership",
		"node_id", e.nodeID,
		"lock_key", lockKey,
	)

	return nil
}

// ReleaseAll releases all held locks. Call during graceful shutdown.
func (e *Elector) ReleaseAll(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for lockKey := range e.locks {
		_, err := e.pool.Exec(ctx, "SELECT pg_advisory_unlock($1)", lockKey)
		if err != nil {
			slog.Error("failed to release lock", "lock_key", lockKey, "error", err)
		}
	}

	e.locks = make(map[int64]bool)
	return nil
}

// IsLeader returns true if this node holds the specified lock.
func (e *Elector) IsLeader(lockKey int64) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.locks[lockKey]
}

// NodeID returns the node identifier.
func (e *Elector) NodeID() string {
	return e.nodeID
}
