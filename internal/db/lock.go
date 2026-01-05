package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// LockConfig holds configuration for database locking operations.
type LockConfig struct {
	Timeout       time.Duration
	RetryInterval time.Duration
	MaxRetries    int
}

// ErrLockTimeout is returned when a lock cannot be acquired after all retries.
var ErrLockTimeout = errors.New("failed to acquire lock after maximum retries")

// WithLock executes a function within a transaction with row-level locking.
// It sets the PostgreSQL lock_timeout and retries on lock conflicts.
func (db *DB) WithLock(ctx context.Context, cfg LockConfig, fn func(tx pgx.Tx) error) error {
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			slog.Debug("retrying lock acquisition",
				"attempt", attempt,
				"max_retries", cfg.MaxRetries,
				"retry_interval", cfg.RetryInterval,
			)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(cfg.RetryInterval):
			}
		}

		err := db.executeWithLock(ctx, cfg.Timeout, fn)
		if err == nil {
			return nil
		}

		// Check if it's a lock timeout error (PostgreSQL error code 55P03)
		if isLockTimeoutError(err) {
			lastErr = err
			continue
		}

		// For other errors, return immediately
		return err
	}

	return fmt.Errorf("%w: %v", ErrLockTimeout, lastErr)
}

// executeWithLock runs the function in a transaction with lock_timeout set.
func (db *DB) executeWithLock(ctx context.Context, timeout time.Duration, fn func(tx pgx.Tx) error) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Set lock_timeout for this transaction (in milliseconds)
	timeoutMs := timeout.Milliseconds()
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL lock_timeout = '%dms'", timeoutMs)); err != nil {
		return fmt.Errorf("setting lock timeout: %w", err)
	}

	// Execute the function
	if err := fn(tx); err != nil {
		return err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// isLockTimeoutError checks if the error is a PostgreSQL lock timeout error.
// PostgreSQL error code 55P03 = lock_not_available
func isLockTimeoutError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "55P03"
	}
	return false
}

// IsSerializationError checks if the error is a PostgreSQL serialization failure.
// PostgreSQL error code 40001 = serialization_failure
func IsSerializationError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "40001"
	}
	return false
}
