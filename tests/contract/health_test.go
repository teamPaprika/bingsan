package contract

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kimuyb/bingsan/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthEndpoints(t *testing.T) {
	// Create test server
	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		checkBody      bool
	}{
		{
			name:           "GET /health returns 200",
			method:         "GET",
			path:           "/health",
			expectedStatus: http.StatusOK,
			checkBody:      true,
		},
		{
			name:           "GET /ready returns 200 or 503 without DB",
			method:         "GET",
			path:           "/ready",
			expectedStatus: http.StatusServiceUnavailable, // No DB connected
			checkBody:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)

			resp, err := server.App().Test(req, -1)
			require.NoError(t, err, "Request execution failed")
			defer resp.Body.Close()

			// Check status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Unexpected status code")

			if tc.checkBody {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err, "Failed to read response body")
				assert.NotEmpty(t, body, "Response body should not be empty")
			}
		})
	}
}

func TestHealthCheckResponseFormat(t *testing.T) {
	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := server.App().Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Health endpoint should return JSON
	contentType := resp.Header.Get("Content-Type")
	assert.Contains(t, contentType, "application/json", "Health endpoint should return JSON")
}

func TestMetricsEndpoint(t *testing.T) {
	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := server.App().Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Metrics endpoint should return 200")

	// Metrics should return Prometheus text format
	contentType := resp.Header.Get("Content-Type")
	assert.Contains(t, contentType, "text/plain", "Metrics endpoint should return text/plain")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Should contain at least some Prometheus metrics
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "# HELP", "Should contain Prometheus HELP comments")
	assert.Contains(t, bodyStr, "# TYPE", "Should contain Prometheus TYPE comments")
}
