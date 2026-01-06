package leader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLockKeyConstants(t *testing.T) {
	// Ensure lock keys are unique and non-zero
	assert.NotEqual(t, int64(0), LockKeyCleanup)
	assert.NotEqual(t, int64(0), LockKeyMetrics)
	assert.NotEqual(t, LockKeyCleanup, LockKeyMetrics)
}

func TestNewElector(t *testing.T) {
	elector := NewElector(nil, "test-node")
	require.NotNil(t, elector)
	assert.Equal(t, "test-node", elector.nodeID)
	assert.NotNil(t, elector.locks)
	assert.Empty(t, elector.locks)
}

func TestElector_NodeID(t *testing.T) {
	elector := NewElector(nil, "my-node-id")
	assert.Equal(t, "my-node-id", elector.NodeID())
}

func TestElector_IsLeader(t *testing.T) {
	elector := NewElector(nil, "node1")

	t.Run("returns false when not holding lock", func(t *testing.T) {
		assert.False(t, elector.IsLeader(LockKeyCleanup))
	})

	t.Run("returns true when holding lock", func(t *testing.T) {
		// Manually set lock to simulate acquired state
		elector.locks[LockKeyCleanup] = true
		assert.True(t, elector.IsLeader(LockKeyCleanup))
	})

	t.Run("different locks are independent", func(t *testing.T) {
		elector.locks[LockKeyCleanup] = true
		assert.True(t, elector.IsLeader(LockKeyCleanup))
		assert.False(t, elector.IsLeader(LockKeyMetrics))
	})
}

func TestElector_ConcurrentAccess(t *testing.T) {
	elector := NewElector(nil, "node1")

	// Simulate concurrent reads of IsLeader
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = elector.IsLeader(LockKeyCleanup)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
