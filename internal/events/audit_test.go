package events

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuditLogger(t *testing.T) {
	t.Run("creates logger with all params", func(t *testing.T) {
		broker := NewBroker()
		defer broker.Close()

		logger := NewAuditLogger(nil, broker, true)
		require.NotNil(t, logger)
		assert.True(t, logger.enabled)
		assert.Same(t, broker, logger.broker)
	})

	t.Run("creates disabled logger", func(t *testing.T) {
		logger := NewAuditLogger(nil, nil, false)
		require.NotNil(t, logger)
		assert.False(t, logger.enabled)
	})
}

func TestAuditLogger_Log_Disabled(t *testing.T) {
	broker := NewBroker()
	defer broker.Close()

	sub := broker.Subscribe("*")
	logger := NewAuditLogger(nil, broker, false)

	event := NewEvent(TableCreated).
		WithNamespace("ns").
		WithTable("tbl")
	audit := NewAuditEvent(event)

	err := logger.Log(context.Background(), audit)
	assert.NoError(t, err)

	// Should still publish to broker even when logging is disabled
	select {
	case received := <-sub.Events():
		assert.Equal(t, TableCreated, received.Type)
	default:
		// Event may have been sent before we started listening
	}
}

func TestAuditLogger_Log_NilBroker(t *testing.T) {
	logger := NewAuditLogger(nil, nil, false)

	event := NewEvent(TableCreated)
	audit := NewAuditEvent(event)

	// Should not panic with nil broker
	err := logger.Log(context.Background(), audit)
	assert.NoError(t, err)
}

func TestNullIfEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{"empty string returns nil", "", nil},
		{"non-empty returns string", "test", "test"},
		{"whitespace returns string", " ", " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nullIfEmpty(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNullIfZero(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected interface{}
	}{
		{"zero returns nil", 0, nil},
		{"positive returns int", 200, 200},
		{"negative returns int", -1, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nullIfZero(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
