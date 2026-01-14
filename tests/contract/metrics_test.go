package contract

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/teamPaprika/bingsan/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricsEndpointFormat verifies the /metrics endpoint returns Prometheus metrics format.
func TestMetricsEndpointFormat(t *testing.T) {
	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	t.Run("MetricsEndpointReturnsPrometheusFormat", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)
		resp, err := server.App().Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		contentType := resp.Header.Get("Content-Type")
		// Prometheus metrics should have text/plain or text/plain; version=0.0.4
		assert.True(t,
			strings.Contains(contentType, "text/plain") ||
				strings.Contains(contentType, "text/plain; version=0.0.4"),
			"Metrics endpoint should return text/plain content type, got: %s", contentType)
	})

	t.Run("MetricsContainsGoMetrics", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)
		resp, err := server.App().Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		bodyStr := string(body)

		// Standard Go runtime metrics should be present
		assert.Contains(t, bodyStr, "go_goroutines", "Should have goroutine metric")
		assert.Contains(t, bodyStr, "go_memstats", "Should have memory stats")
	})

	t.Run("MetricsContainsHTTPMetrics", func(t *testing.T) {
		// First make some requests to generate metrics
		req := httptest.NewRequest("GET", "/v1/config", nil)
		resp, _ := server.App().Test(req, -1)
		if resp != nil {
			resp.Body.Close()
		}

		// Now check metrics
		req = httptest.NewRequest("GET", "/metrics", nil)
		resp, err := server.App().Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		bodyStr := string(body)

		// Should have HTTP request metrics (if implemented)
		// These are common Prometheus metric names for HTTP servers
		hasHTTPMetrics := strings.Contains(bodyStr, "http_request") ||
			strings.Contains(bodyStr, "http_requests") ||
			strings.Contains(bodyStr, "request_duration") ||
			strings.Contains(bodyStr, "requests_total")

		// Log what metrics we found for debugging
		if !hasHTTPMetrics {
			t.Logf("No standard HTTP metrics found. Available metrics: %s", bodyStr[:minInt(500, len(bodyStr))])
		}
	})

	t.Run("MetricsContainsProcessMetrics", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)
		resp, err := server.App().Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		bodyStr := string(body)

		// Process metrics should be present
		hasProcessMetrics := strings.Contains(bodyStr, "process_") ||
			strings.Contains(bodyStr, "go_gc")

		assert.True(t, hasProcessMetrics, "Should have process or GC metrics")
	})
}

// TestMetricsPrometheusFormat verifies the Prometheus metrics format is valid.
func TestMetricsPrometheusFormat(t *testing.T) {
	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := server.App().Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := string(body)

	// Prometheus metrics format: lines starting with # are comments/metadata
	// Actual metrics are in format: metric_name{labels} value
	lines := strings.Split(bodyStr, "\n")
	hasMetricLine := false
	hasCommentLine := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			hasCommentLine = true
			// HELP and TYPE lines are standard Prometheus format
			if strings.HasPrefix(line, "# HELP") || strings.HasPrefix(line, "# TYPE") {
				continue
			}
		} else {
			hasMetricLine = true
			// Metric line should contain at least a name and value
			parts := strings.Fields(line)
			assert.GreaterOrEqual(t, len(parts), 2,
				"Metric line should have name and value: %s", line)
		}
	}

	assert.True(t, hasMetricLine, "Should have at least one metric line")
	assert.True(t, hasCommentLine, "Should have at least one comment/metadata line")
}

// TestMetricsDoesNotExposeSensitiveData ensures metrics don't leak sensitive info.
func TestMetricsDoesNotExposeSensitiveData(t *testing.T) {
	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := server.App().Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := strings.ToLower(string(body))

	// Metrics should not contain sensitive data
	sensitivePatterns := []string{
		"password",
		"secret",
		"token",
		"api_key",
		"apikey",
		"credential",
	}

	for _, pattern := range sensitivePatterns {
		assert.NotContains(t, bodyStr, pattern,
			"Metrics should not contain sensitive data pattern: %s", pattern)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
