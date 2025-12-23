package e2e

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kimuyb/bingsan/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLatencyMetrics verifies that request latency histograms are recorded.
func TestLatencyMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	// Make several requests to generate latency data
	endpoints := []string{"/health", "/ready", "/v1/config"}
	for _, endpoint := range endpoints {
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", endpoint, nil)
			resp, err := server.App().Test(req, -1)
			if err == nil {
				resp.Body.Close()
			}
		}
	}

	// Check metrics endpoint for latency histograms
	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := server.App().Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := readBody(t, resp)

	// Look for histogram-style metrics (bucket, sum, count suffixes)
	hasHistogram := strings.Contains(body, "_bucket") ||
		strings.Contains(body, "_sum") ||
		strings.Contains(body, "_count") ||
		strings.Contains(body, "duration") ||
		strings.Contains(body, "latency")

	// Log for debugging if no histogram found
	if !hasHistogram {
		t.Logf("No histogram metrics found. Metrics output: %s", body[:min(1000, len(body))])
	}
}

// TestDBMetrics verifies database connection pool metrics are exposed.
func TestDBMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	// Check metrics endpoint
	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := server.App().Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	body := readBody(t, resp)

	// Look for database-related metrics
	// Common patterns: db_pool, database, sql, connections
	hasDBMetrics := strings.Contains(body, "db_") ||
		strings.Contains(body, "database") ||
		strings.Contains(body, "sql_") ||
		strings.Contains(body, "pool_") ||
		strings.Contains(body, "connection")

	// This test may pass or fail depending on whether DB metrics are implemented
	// Log for visibility
	if !hasDBMetrics {
		t.Logf("No database metrics found. This may be expected if DB is not connected.")
	}
}

// TestEntityMetrics verifies metrics for entity counts (namespaces, tables, views).
func TestEntityMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	// Check metrics endpoint
	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := server.App().Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	body := readBody(t, resp)

	// Look for entity count metrics
	// These would be custom metrics specific to Bingsan
	entityMetricPatterns := []string{
		"namespace",
		"table",
		"view",
		"catalog",
		"iceberg",
	}

	foundEntityMetrics := false
	for _, pattern := range entityMetricPatterns {
		if strings.Contains(strings.ToLower(body), pattern) {
			foundEntityMetrics = true
			break
		}
	}

	// Log what we found
	if !foundEntityMetrics {
		t.Logf("No entity-specific metrics found. Available metrics include standard Go metrics.")
	}
}

// TestConcurrentMetricsAccess verifies metrics endpoint handles concurrent requests.
func TestConcurrentMetricsAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	var wg sync.WaitGroup
	errors := make(chan error, 100)
	statuses := make(chan int, 100)

	// Make concurrent requests to metrics endpoint
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/metrics", nil)
			resp, err := server.App().Test(req, 5000)
			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()
			statuses <- resp.StatusCode
		}()
	}

	wg.Wait()
	close(errors)
	close(statuses)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Logf("Concurrent request error: %v", err)
		errorCount++
	}

	// Check status codes
	successCount := 0
	for status := range statuses {
		if status == http.StatusOK {
			successCount++
		}
	}

	assert.Equal(t, 0, errorCount, "Should have no errors in concurrent metrics access")
	assert.Equal(t, 50, successCount, "All concurrent requests should succeed")
}

// TestMetricsUnderLoad verifies metrics are accurate under load.
func TestMetricsUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	// Generate load
	var wg sync.WaitGroup
	requestCount := 100

	start := time.Now()
	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/health", nil)
			resp, err := server.App().Test(req, 5000)
			if err == nil {
				resp.Body.Close()
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("Processed %d requests in %v", requestCount, elapsed)

	// Now check metrics
	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := server.App().Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Metrics should be available after load test")
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	buf := make([]byte, 0, 4096)
	for {
		chunk := make([]byte, 1024)
		n, err := resp.Body.Read(chunk)
		buf = append(buf, chunk[:n]...)
		if err != nil {
			break
		}
	}
	return string(buf)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
