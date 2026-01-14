package contract

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/teamPaprika/bingsan/internal/api"
	"github.com/teamPaprika/bingsan/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrorResponses tests that error responses match the Iceberg error schema.
func TestErrorResponses(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	spec, err := LoadSpec()
	require.NoError(t, err, "Failed to load OpenAPI spec")

	cfg := testConfigWithDB()
	database, err := db.New(cfg.Database)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}
	defer database.Close()

	server := api.NewServer(cfg, database)

	t.Run("400_InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/v1/namespaces", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := server.App().Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "error", "Response should contain error field")
	})

	t.Run("400_MissingRequiredField", func(t *testing.T) {
		// Missing 'namespace' field
		reqBody := []byte(`{"properties": {}}`)

		req := httptest.NewRequest("POST", "/v1/namespaces", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := server.App().Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "error", "Response should contain error field")
	})

	t.Run("404_NamespaceNotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/namespaces/does_not_exist_12345", nil)
		req.Header.Set("Content-Type", "application/json")

		route, pathParams, err := spec.Router.FindRoute(req)
		require.NoError(t, err, "Route not found in spec")

		requestInput := &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		}
		require.NoError(t, openapi3filter.ValidateRequest(context.Background(), requestInput))

		resp, err := server.App().Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		validateResponse(t, requestInput, resp.StatusCode, resp.Header, body)
	})

	t.Run("404_TableNotFound", func(t *testing.T) {
		// Create namespace first
		createNamespace(t, server.App(), "error_test_ns")
		defer cleanupNamespace(server.App(), "error_test_ns")

		req := httptest.NewRequest("GET", "/v1/namespaces/error_test_ns/tables/does_not_exist", nil)
		req.Header.Set("Content-Type", "application/json")

		route, pathParams, err := spec.Router.FindRoute(req)
		require.NoError(t, err, "Route not found in spec")

		requestInput := &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		}
		require.NoError(t, openapi3filter.ValidateRequest(context.Background(), requestInput))

		resp, err := server.App().Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		validateResponse(t, requestInput, resp.StatusCode, resp.Header, body)
	})

	t.Run("409_NamespaceAlreadyExists", func(t *testing.T) {
		// Create namespace
		createNamespace(t, server.App(), "conflict_test_ns")
		defer cleanupNamespace(server.App(), "conflict_test_ns")

		// Try to create it again
		reqBody := []byte(`{"namespace": ["conflict_test_ns"]}`)
		req := httptest.NewRequest("POST", "/v1/namespaces", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := server.App().Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "error", "Response should contain error field")
	})

	t.Run("409_CannotDeleteNonEmptyNamespace", func(t *testing.T) {
		// Create namespace with a table
		createNamespace(t, server.App(), "nonempty_test_ns")
		createTable(t, server.App(), "nonempty_test_ns", "test_table")
		defer func() {
			cleanupTable(server.App(), "nonempty_test_ns", "test_table")
			cleanupNamespace(server.App(), "nonempty_test_ns")
		}()

		// Try to delete non-empty namespace
		req := httptest.NewRequest("DELETE", "/v1/namespaces/nonempty_test_ns", nil)

		resp, err := server.App().Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return 409 Conflict because namespace is not empty
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "error", "Response should contain error field")
	})
}

// TestIcebergErrorFormat verifies that error responses match the Iceberg error schema.
func TestIcebergErrorFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := testConfigWithDB()
	database, err := db.New(cfg.Database)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}
	defer database.Close()

	server := api.NewServer(cfg, database)

	req := httptest.NewRequest("GET", "/v1/namespaces/nonexistent_ns_12345", nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := server.App().Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Iceberg error format should have: error.message, error.type, error.code
	// Check that the response is valid JSON with error structure
	assert.Contains(t, string(body), "error", "Response should contain 'error' key")
	assert.Contains(t, string(body), "message", "Error should contain 'message' field")
	assert.Contains(t, string(body), "type", "Error should contain 'type' field")
}
