package e2e

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kimuyb/bingsan/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDatabaseConnectionFailure verifies behavior when database is unavailable.
func TestDatabaseConnectionFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Create server without database connection
	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	t.Run("HealthEndpointWithoutDB", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		resp, err := server.App().Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Health endpoint should still return 200 (liveness)
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Health endpoint should be available")
	})

	t.Run("ReadyEndpointWithoutDB", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ready", nil)
		resp, err := server.App().Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Ready endpoint should return 503 when DB is unavailable
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode, "Ready endpoint should indicate unavailable")
	})

	t.Run("APIEndpointsWithoutDB", func(t *testing.T) {
		// Attempting to list namespaces should fail gracefully
		req := httptest.NewRequest("GET", "/v1/namespaces", nil)
		req.Header.Set("Content-Type", "application/json")
		resp, err := server.App().Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return an error, not crash
		// The exact status depends on implementation - could be 500 or 503
		assert.True(t, resp.StatusCode >= 400, "Should return error status when DB unavailable")
	})

	t.Run("MetricsEndpointWithoutDB", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)
		resp, err := server.App().Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Metrics endpoint should still be available
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Metrics endpoint should be available")
	})
}

// TestServerRecovery verifies that the server recovers from panics.
func TestServerRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	// Make a request that might cause issues (invalid path)
	req := httptest.NewRequest("GET", "/v1/invalid/path/that/does/not/exist", nil)
	resp, err := server.App().Test(req, -1)
	require.NoError(t, err, "Server should not crash on invalid path")
	resp.Body.Close()

	// Server should still be able to handle new requests
	req2 := httptest.NewRequest("GET", "/health", nil)
	resp2, err := server.App().Test(req2, -1)
	require.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode, "Server should recover and handle new requests")
}

// TestErrorResponseFormat verifies that error responses are properly formatted.
func TestErrorResponseFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	// Request a non-existent endpoint
	req := httptest.NewRequest("GET", "/v1/nonexistent", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.App().Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Error response should be JSON
	contentType := resp.Header.Get("Content-Type")
	assert.Contains(t, contentType, "application/json", "Error response should be JSON")

	// Body should contain error information
	bodyStr := string(body)
	assert.NotEmpty(t, bodyStr, "Error response should have body")
}
