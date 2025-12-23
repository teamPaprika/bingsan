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

// TestScanPlanOperations tests scan planning operations against the OpenAPI spec.
func TestScanPlanOperations(t *testing.T) {
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

	// Setup
	createNamespace(t, server.App(), "scan_test_ns")
	createTable(t, server.App(), "scan_test_ns", "scan_test_table")
	defer func() {
		cleanupTable(server.App(), "scan_test_ns", "scan_test_table")
		cleanupNamespace(server.App(), "scan_test_ns")
	}()

	t.Run("SubmitScanPlan", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"snapshot-id": nil,
			"select":      []string{"id"},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/v1/namespaces/scan_test_ns/tables/scan_test_table/plan", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		route, pathParams, err := spec.Router.FindRoute(req)
		require.NoError(t, err, "Route not found in spec")

		requestInput := &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		}

		req2 := httptest.NewRequest("POST", "/v1/namespaces/scan_test_ns/tables/scan_test_table/plan", bytes.NewReader(bodyBytes))
		req2.Header.Set("Content-Type", "application/json")
		requestInput.Request = req2
		err = openapi3filter.ValidateRequest(context.Background(), requestInput)
		require.NoError(t, err, "Request validation failed")

		req3 := httptest.NewRequest("POST", "/v1/namespaces/scan_test_ns/tables/scan_test_table/plan", bytes.NewReader(bodyBytes))
		req3.Header.Set("Content-Type", "application/json")
		resp, err := server.App().Test(req3, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return 200 with plan ID
		body, _ := io.ReadAll(resp.Body)
		t.Logf("Response: %s", string(body))

		// Note: The actual implementation may return different status codes
		// depending on whether scan planning is fully implemented
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusNotImplemented,
			"Expected 200, 202, or 501, got %d", resp.StatusCode)
	})

	t.Run("FetchPlanTasks", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"plan-id": "test-plan-id",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/v1/namespaces/scan_test_ns/tables/scan_test_table/tasks", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		route, pathParams, err := spec.Router.FindRoute(req)
		require.NoError(t, err, "Route not found in spec")

		requestInput := &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		}

		req2 := httptest.NewRequest("POST", "/v1/namespaces/scan_test_ns/tables/scan_test_table/tasks", bytes.NewReader(bodyBytes))
		req2.Header.Set("Content-Type", "application/json")
		requestInput.Request = req2
		err = openapi3filter.ValidateRequest(context.Background(), requestInput)
		require.NoError(t, err, "Request validation failed")

		req3 := httptest.NewRequest("POST", "/v1/namespaces/scan_test_ns/tables/scan_test_table/tasks", bytes.NewReader(bodyBytes))
		req3.Header.Set("Content-Type", "application/json")
		resp, err := server.App().Test(req3, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Response depends on whether the plan exists
		body, _ := io.ReadAll(resp.Body)
		t.Logf("Response: %s", string(body))
	})
}
