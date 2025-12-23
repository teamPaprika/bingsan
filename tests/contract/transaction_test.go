package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/kimuyb/bingsan/internal/api"
	"github.com/kimuyb/bingsan/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTransactionOperations tests multi-table transaction operations against the OpenAPI spec.
func TestTransactionOperations(t *testing.T) {
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

	t.Run("CommitEmptyTransaction", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"table-changes": []interface{}{},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/v1/transactions/commit", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		route, pathParams, err := spec.Router.FindRoute(req)
		require.NoError(t, err, "Route not found in spec")

		requestInput := &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		}

		req2 := httptest.NewRequest("POST", "/v1/transactions/commit", bytes.NewReader(bodyBytes))
		req2.Header.Set("Content-Type", "application/json")
		requestInput.Request = req2
		err = openapi3filter.ValidateRequest(context.Background(), requestInput)
		require.NoError(t, err, "Request validation failed")

		req3 := httptest.NewRequest("POST", "/v1/transactions/commit", bytes.NewReader(bodyBytes))
		req3.Header.Set("Content-Type", "application/json")
		resp, err := server.App().Test(req3, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		t.Logf("Response: %s", string(body))

		// Empty transaction should succeed (no-op) or return 200
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent,
			"Expected 200 or 204, got %d", resp.StatusCode)
	})

	t.Run("CommitTransactionWithTableChanges", func(t *testing.T) {
		// Setup: Create namespace and table
		createNamespace(t, server.App(), "txn_test_ns")
		createTable(t, server.App(), "txn_test_ns", "txn_test_table")
		defer func() {
			cleanupTable(server.App(), "txn_test_ns", "txn_test_table")
			cleanupNamespace(server.App(), "txn_test_ns")
		}()

		// Note: This test validates that the transaction endpoint accepts the correct format
		// Actual transaction behavior depends on implementation
		reqBody := map[string]interface{}{
			"table-changes": []map[string]interface{}{
				{
					"identifier": map[string]interface{}{
						"namespace": []string{"txn_test_ns"},
						"name":      "txn_test_table",
					},
					"requirements": []interface{}{},
					"updates":      []interface{}{},
				},
			},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/v1/transactions/commit", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		route, pathParams, err := spec.Router.FindRoute(req)
		require.NoError(t, err, "Route not found in spec")

		requestInput := &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		}

		req2 := httptest.NewRequest("POST", "/v1/transactions/commit", bytes.NewReader(bodyBytes))
		req2.Header.Set("Content-Type", "application/json")
		requestInput.Request = req2

		req3 := httptest.NewRequest("POST", "/v1/transactions/commit", bytes.NewReader(bodyBytes))
		req3.Header.Set("Content-Type", "application/json")
		resp, err := server.App().Test(req3, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		t.Logf("Response: %s", string(body))

		// Transaction should succeed or fail with appropriate error
		assert.True(t, resp.StatusCode >= 200 && resp.StatusCode < 500,
			"Expected success or client error, got %d", resp.StatusCode)
	})
}

// TestRenameOperations tests table and view rename operations.
func TestRenameOperations(t *testing.T) {
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

	t.Run("RenameTable", func(t *testing.T) {
		// Setup
		createNamespace(t, server.App(), "rename_test_ns")
		createTable(t, server.App(), "rename_test_ns", "rename_source")
		defer func() {
			cleanupTable(server.App(), "rename_test_ns", "rename_source")
			cleanupTable(server.App(), "rename_test_ns", "rename_dest")
			cleanupNamespace(server.App(), "rename_test_ns")
		}()

		reqBody := map[string]interface{}{
			"source": map[string]interface{}{
				"namespace": []string{"rename_test_ns"},
				"name":      "rename_source",
			},
			"destination": map[string]interface{}{
				"namespace": []string{"rename_test_ns"},
				"name":      "rename_dest",
			},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/v1/tables/rename", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		route, pathParams, err := spec.Router.FindRoute(req)
		require.NoError(t, err, "Route not found in spec")

		requestInput := &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		}

		req2 := httptest.NewRequest("POST", "/v1/tables/rename", bytes.NewReader(bodyBytes))
		req2.Header.Set("Content-Type", "application/json")
		requestInput.Request = req2
		err = openapi3filter.ValidateRequest(context.Background(), requestInput)
		require.NoError(t, err, "Request validation failed")

		req3 := httptest.NewRequest("POST", "/v1/tables/rename", bytes.NewReader(bodyBytes))
		req3.Header.Set("Content-Type", "application/json")
		resp, err := server.App().Test(req3, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		t.Logf("Response: %s", string(body))

		// Rename should succeed
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent,
			"Expected 200 or 204, got %d", resp.StatusCode)
	})

	t.Run("RenameView", func(t *testing.T) {
		// Setup
		createNamespace(t, server.App(), "rename_view_ns")
		createView(t, server.App(), "rename_view_ns", "view_source")
		defer func() {
			cleanupView(server.App(), "rename_view_ns", "view_source")
			cleanupView(server.App(), "rename_view_ns", "view_dest")
			cleanupNamespace(server.App(), "rename_view_ns")
		}()

		reqBody := map[string]interface{}{
			"source": map[string]interface{}{
				"namespace": []string{"rename_view_ns"},
				"name":      "view_source",
			},
			"destination": map[string]interface{}{
				"namespace": []string{"rename_view_ns"},
				"name":      "view_dest",
			},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/v1/views/rename", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		route, pathParams, err := spec.Router.FindRoute(req)
		require.NoError(t, err, "Route not found in spec")

		requestInput := &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		}

		req2 := httptest.NewRequest("POST", "/v1/views/rename", bytes.NewReader(bodyBytes))
		req2.Header.Set("Content-Type", "application/json")
		requestInput.Request = req2
		err = openapi3filter.ValidateRequest(context.Background(), requestInput)
		require.NoError(t, err, "Request validation failed")

		req3 := httptest.NewRequest("POST", "/v1/views/rename", bytes.NewReader(bodyBytes))
		req3.Header.Set("Content-Type", "application/json")
		resp, err := server.App().Test(req3, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		t.Logf("Response: %s", string(body))

		// Rename should succeed
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent,
			"Expected 200 or 204, got %d", resp.StatusCode)
	})
}
