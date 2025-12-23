// Package tests provides shared test utilities for all test suites.
package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/kimuyb/bingsan/internal/api"
	"github.com/kimuyb/bingsan/internal/config"
	"github.com/kimuyb/bingsan/internal/db"
)

// TestServer wraps an API server for testing.
type TestServer struct {
	Server *api.Server
	Config *config.Config
	DB     *db.DB
	URL    string
}

// NewTestConfig returns a test configuration with defaults suitable for testing.
func NewTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         0, // Random port
			Debug:        true,
			Version:      "test",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			IdleTimeout:  10 * time.Second,
		},
		Database: config.DatabaseConfig{
			Host:            getEnv("TEST_DB_HOST", "localhost"),
			Port:            5432,
			User:            getEnv("TEST_DB_USER", "iceberg"),
			Password:        getEnv("TEST_DB_PASSWORD", "iceberg"),
			Database:        getEnv("TEST_DB_NAME", "iceberg_catalog_test"),
			SSLMode:         "disable",
			MaxOpenConns:    5,
			MaxIdleConns:    2,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 1 * time.Minute,
		},
		Storage: config.StorageConfig{
			Type:      "local",
			Warehouse: "/tmp/iceberg-test/warehouse",
			Local: config.LocalConfig{
				RootPath: "/tmp/iceberg-test/data",
			},
		},
		Auth: config.AuthConfig{
			Enabled: false,
		},
		Catalog: config.CatalogConfig{
			Prefix:            "",
			LockTimeout:       5 * time.Second,
			LockRetryInterval: 50 * time.Millisecond,
			MaxLockRetries:    10,
		},
	}
}

// SetupTestServer creates a new test server instance.
// The caller is responsible for calling Cleanup() when done.
func SetupTestServer(t *testing.T) *TestServer {
	t.Helper()

	cfg := NewTestConfig()

	// Connect to database
	database, err := db.New(cfg.Database)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Create server
	server := api.NewServer(cfg, database)

	// Get the Fiber app for testing
	ts := &TestServer{
		Server: server,
		Config: cfg,
		DB:     database,
	}

	t.Cleanup(func() {
		ts.Cleanup()
	})

	return ts
}

// SetupTestServerWithoutDB creates a test server without database connection.
// Useful for testing endpoints that don't require database access.
func SetupTestServerWithoutDB(t *testing.T) *TestServer {
	t.Helper()

	cfg := NewTestConfig()

	// Create server without database (will use nil)
	server := api.NewServer(cfg, nil)

	ts := &TestServer{
		Server: server,
		Config: cfg,
	}

	t.Cleanup(func() {
		ts.Cleanup()
	})

	return ts
}

// App returns the underlying Fiber app for direct testing.
func (ts *TestServer) App() *fiber.App {
	return ts.Server.App()
}

// Request executes an HTTP request against the test server.
func (ts *TestServer) Request(method, path string, body string) (*http.Response, error) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return ts.App().Test(req, -1)
}

// RequestWithHeaders executes an HTTP request with custom headers.
func (ts *TestServer) RequestWithHeaders(method, path string, body string, headers map[string]string) (*http.Response, error) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return ts.App().Test(req, -1)
}

// Cleanup releases resources used by the test server.
func (ts *TestServer) Cleanup() {
	if ts.DB != nil {
		ts.DB.Close()
	}
}

// CleanupTestNamespace removes all test data from the namespace.
func (ts *TestServer) CleanupTestNamespace(ctx context.Context, namespace string) error {
	if ts.DB == nil {
		return nil
	}
	// Implementation depends on your database schema
	// This is a placeholder - implement based on your actual schema
	return nil
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// MustParseJSON parses JSON or fails the test.
func MustParseJSON(t *testing.T, data []byte, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
}

// ReadBody reads and returns the response body.
func ReadBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	return body
}

// AssertStatusCode checks that the response has the expected status code.
func AssertStatusCode(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Errorf("Expected status code %d, got %d", expected, resp.StatusCode)
	}
}

// AssertContentType checks that the response has the expected content type.
func AssertContentType(t *testing.T, resp *http.Response, expected string) {
	t.Helper()
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, expected) {
		t.Errorf("Expected content type %s, got %s", expected, ct)
	}
}

// WaitForServer waits for the server to be ready.
func WaitForServer(baseURL string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("server not ready after %v", timeout)
		case <-ticker.C:
			resp, err := http.Get(baseURL + "/health")
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				return nil
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}
