package handlers

import (
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPgErrUniqueViolation(t *testing.T) {
	assert.Equal(t, "23505", PgErrUniqueViolation)
}

func TestIsDuplicateError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "unique violation error",
			err:      &pgconn.PgError{Code: "23505"},
			expected: true,
		},
		{
			name:     "different postgres error",
			err:      &pgconn.PgError{Code: "23503"},
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
			name: "wrapped unique violation",
			err: &wrappedError{
				err: &pgconn.PgError{Code: "23505"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDuplicateError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAlreadyExistsError(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return alreadyExistsError(c, "Namespace", "test_ns")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, 409, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "Namespace already exists: test_ns")
	assert.Contains(t, string(body), "AlreadyExistsException")
	assert.Contains(t, string(body), "409")
}

func TestGetStringFromMap(t *testing.T) {
	tests := []struct {
		name       string
		m          map[string]any
		key        string
		expected   string
		expectedOk bool
	}{
		{
			name:       "valid string key",
			m:          map[string]any{"key1": "value1"},
			key:        "key1",
			expected:   "value1",
			expectedOk: true,
		},
		{
			name:       "key not found",
			m:          map[string]any{"key1": "value1"},
			key:        "key2",
			expected:   "",
			expectedOk: false,
		},
		{
			name:       "nil map",
			m:          nil,
			key:        "key1",
			expected:   "",
			expectedOk: false,
		},
		{
			name:       "non-string value",
			m:          map[string]any{"key1": 123},
			key:        "key1",
			expected:   "",
			expectedOk: false,
		},
		{
			name:       "empty string value",
			m:          map[string]any{"key1": ""},
			key:        "key1",
			expected:   "",
			expectedOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := getStringFromMap(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.expectedOk, ok)
		})
	}
}

func TestValidatePageSize(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"zero returns 1", 0, 1},
		{"negative returns 1", -10, 1},
		{"valid value unchanged", 50, 50},
		{"max value unchanged", 1000, 1000},
		{"above max clamped", 2000, 1000},
		{"minimum valid", 1, 1},
		{"mid range", 500, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validatePageSize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
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
