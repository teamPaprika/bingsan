package db

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestLockConfig(t *testing.T) {
	cfg := LockConfig{
		Timeout:       30 * time.Second,
		RetryInterval: 100 * time.Millisecond,
		MaxRetries:    10,
	}

	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.Equal(t, 100*time.Millisecond, cfg.RetryInterval)
	assert.Equal(t, 10, cfg.MaxRetries)
}

func TestErrLockTimeout(t *testing.T) {
	assert.Equal(t, "failed to acquire lock after maximum retries", ErrLockTimeout.Error())
	assert.True(t, errors.Is(ErrLockTimeout, ErrLockTimeout))
}

func TestIsLockTimeoutError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "lock timeout error 55P03",
			err:      &pgconn.PgError{Code: "55P03"},
			expected: true,
		},
		{
			name:     "different postgres error",
			err:      &pgconn.PgError{Code: "23505"},
			expected: false,
		},
		{
			name:     "serialization error 40001",
			err:      &pgconn.PgError{Code: "40001"},
			expected: false,
		},
		{
			name:     "non-postgres error",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name: "wrapped lock timeout error",
			err: &wrappedError{
				err: &pgconn.PgError{Code: "55P03"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLockTimeoutError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSerializationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "serialization error 40001",
			err:      &pgconn.PgError{Code: "40001"},
			expected: true,
		},
		{
			name:     "lock timeout error 55P03",
			err:      &pgconn.PgError{Code: "55P03"},
			expected: false,
		},
		{
			name:     "different postgres error",
			err:      &pgconn.PgError{Code: "23505"},
			expected: false,
		},
		{
			name:     "non-postgres error",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name: "wrapped serialization error",
			err: &wrappedError{
				err: &pgconn.PgError{Code: "40001"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSerializationError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPgErrorCodes(t *testing.T) {
	t.Run("lock timeout code", func(t *testing.T) {
		err := &pgconn.PgError{Code: "55P03"}
		assert.True(t, isLockTimeoutError(err))
		assert.False(t, IsSerializationError(err))
	})

	t.Run("serialization failure code", func(t *testing.T) {
		err := &pgconn.PgError{Code: "40001"}
		assert.False(t, isLockTimeoutError(err))
		assert.True(t, IsSerializationError(err))
	})

	t.Run("unique violation code", func(t *testing.T) {
		err := &pgconn.PgError{Code: "23505"}
		assert.False(t, isLockTimeoutError(err))
		assert.False(t, IsSerializationError(err))
	})

	t.Run("deadlock detected code", func(t *testing.T) {
		err := &pgconn.PgError{Code: "40P01"}
		assert.False(t, isLockTimeoutError(err))
		assert.False(t, IsSerializationError(err))
	})
}

// wrappedError is a helper for testing errors.As behavior
type wrappedError struct {
	err error
}

func (w *wrappedError) Error() string {
	return w.err.Error()
}

func (w *wrappedError) Unwrap() error {
	return w.err
}
