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
	"github.com/gofiber/fiber/v2"
	"github.com/kimuyb/bingsan/internal/api"
	"github.com/kimuyb/bingsan/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTableOperations tests table CRUD operations against the OpenAPI spec.
func TestTableOperations(t *testing.T) {
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

	// Setup: Create a test namespace
	createNamespace(t, server.App(), "table_test_ns")
	defer cleanupNamespace(server.App(), "table_test_ns")

	t.Run("ListTables", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/namespaces/table_test_ns/tables", nil)
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

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		validateResponse(t, requestInput, resp.StatusCode, resp.Header, body)
	})

	t.Run("CreateTable", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "contract_test_table",
			"schema": map[string]interface{}{
				"schema-id": 0,
				"type":      "struct",
				"fields": []map[string]interface{}{
					{"id": 1, "name": "id", "type": "long", "required": true},
					{"id": 2, "name": "name", "type": "string", "required": false},
				},
			},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/v1/namespaces/table_test_ns/tables", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		route, pathParams, err := spec.Router.FindRoute(req)
		require.NoError(t, err, "Route not found in spec")

		requestInput := &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		}

		req2 := httptest.NewRequest("POST", "/v1/namespaces/table_test_ns/tables", bytes.NewReader(bodyBytes))
		req2.Header.Set("Content-Type", "application/json")
		requestInput.Request = req2
		err = openapi3filter.ValidateRequest(context.Background(), requestInput)
		require.NoError(t, err, "Request validation failed")

		req3 := httptest.NewRequest("POST", "/v1/namespaces/table_test_ns/tables", bytes.NewReader(bodyBytes))
		req3.Header.Set("Content-Type", "application/json")
		resp, err := server.App().Test(req3, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should be 200 (created) or 409 (already exists)
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusConflict,
			"Expected 200 or 409, got %d", resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusOK {
			validateResponse(t, requestInput, resp.StatusCode, resp.Header, body)
		}

		// Cleanup
		cleanupTable(server.App(), "table_test_ns", "contract_test_table")
	})

	t.Run("LoadTable", func(t *testing.T) {
		// Create table first
		createTable(t, server.App(), "table_test_ns", "load_test_table")
		defer cleanupTable(server.App(), "table_test_ns", "load_test_table")

		req := httptest.NewRequest("GET", "/v1/namespaces/table_test_ns/tables/load_test_table", nil)
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

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		validateResponse(t, requestInput, resp.StatusCode, resp.Header, body)
	})

	t.Run("HeadTable", func(t *testing.T) {
		createTable(t, server.App(), "table_test_ns", "head_test_table")
		defer cleanupTable(server.App(), "table_test_ns", "head_test_table")

		req := httptest.NewRequest("HEAD", "/v1/namespaces/table_test_ns/tables/head_test_table", nil)

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

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("DropTable", func(t *testing.T) {
		createTable(t, server.App(), "table_test_ns", "drop_test_table")

		req := httptest.NewRequest("DELETE", "/v1/namespaces/table_test_ns/tables/drop_test_table", nil)

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

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}

func createTable(t *testing.T, app *fiber.App, namespace, tableName string) {
	t.Helper()
	reqBody := map[string]interface{}{
		"name": tableName,
		"schema": map[string]interface{}{
			"schema-id": 0,
			"type":      "struct",
			"fields": []map[string]interface{}{
				{"id": 1, "name": "id", "type": "long", "required": true},
			},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/namespaces/"+namespace+"/tables", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp != nil {
		resp.Body.Close()
	}
}

func cleanupTable(app *fiber.App, namespace, tableName string) {
	req := httptest.NewRequest("DELETE", "/v1/namespaces/"+namespace+"/tables/"+tableName, nil)
	resp, _ := app.Test(req, -1)
	if resp != nil {
		resp.Body.Close()
	}
}
