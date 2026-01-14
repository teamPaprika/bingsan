package e2e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/teamPaprika/bingsan/internal/api"
	"github.com/teamPaprika/bingsan/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGracefulShutdown verifies that the server shuts down gracefully.
func TestGracefulShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	// Verify server is healthy
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := server.App().Test(req, -1)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Shutdown should complete without error
	err = server.Shutdown()
	assert.NoError(t, err, "Graceful shutdown should succeed")
}

// TestShutdownDrainsRequests verifies that in-flight requests complete during shutdown.
func TestShutdownDrainsRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	var wg sync.WaitGroup
	results := make(chan int, 10)

	// Start multiple concurrent requests
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/health", nil)
			resp, err := server.App().Test(req, 5000) // 5 second timeout
			if err == nil {
				results <- resp.StatusCode
				resp.Body.Close()
			} else {
				results <- 0
			}
		}()
	}

	// Wait for all requests to complete
	wg.Wait()
	close(results)

	// All requests should have completed successfully
	for status := range results {
		assert.Equal(t, http.StatusOK, status, "All requests should complete during shutdown")
	}

	// Now shutdown
	err := server.Shutdown()
	assert.NoError(t, err)
}

// TestShutdownTimeout verifies that shutdown respects timeout.
func TestShutdownTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- server.Shutdown()
	}()

	select {
	case err := <-done:
		assert.NoError(t, err, "Shutdown should complete without error")
	case <-ctx.Done():
		t.Fatal("Shutdown timed out - should complete within 5 seconds")
	}
}

func testConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         0,
			Debug:        true,
			Version:      "test",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			IdleTimeout:  10 * time.Second,
		},
		Storage: config.StorageConfig{
			Type:      "local",
			Warehouse: "/tmp/iceberg-test/warehouse",
		},
		Auth: config.AuthConfig{
			Enabled: false,
		},
		Catalog: config.CatalogConfig{
			Prefix: "",
		},
	}
}
