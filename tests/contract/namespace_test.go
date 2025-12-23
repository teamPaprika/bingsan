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
	"github.com/kimuyb/bingsan/internal/config"
	"github.com/kimuyb/bingsan/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNamespaceOperations tests namespace CRUD operations against the OpenAPI spec.
// These tests require a database connection.
func TestNamespaceOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load OpenAPI spec
	spec, err := LoadSpec()
	require.NoError(t, err, "Failed to load OpenAPI spec")

	// Setup test database
	cfg := testConfigWithDB()
	database, err := db.New(cfg.Database)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}
	defer database.Close()

	server := api.NewServer(cfg, database)

	t.Run("ListNamespaces", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/namespaces", nil)
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

	t.Run("CreateNamespace", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"namespace":  []string{"contract_test_ns"},
			"properties": map[string]string{"owner": "test"},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/v1/namespaces", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		route, pathParams, err := spec.Router.FindRoute(req)
		require.NoError(t, err, "Route not found in spec")

		requestInput := &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		}
		// Re-create request for validation (body already consumed)
		req2 := httptest.NewRequest("POST", "/v1/namespaces", bytes.NewReader(bodyBytes))
		req2.Header.Set("Content-Type", "application/json")
		requestInput.Request = req2

		err = openapi3filter.ValidateRequest(context.Background(), requestInput)
		require.NoError(t, err, "Request validation failed")

		// Execute with original body
		req3 := httptest.NewRequest("POST", "/v1/namespaces", bytes.NewReader(bodyBytes))
		req3.Header.Set("Content-Type", "application/json")
		resp, err := server.App().Test(req3, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should be 200 (created) or 409 (already exists)
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusConflict,
			"Expected 200 or 409, got %d", resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		validateResponse(t, requestInput, resp.StatusCode, resp.Header, body)

		// Cleanup
		cleanupNamespace(server.App(), "contract_test_ns")
	})

	t.Run("GetNamespace", func(t *testing.T) {
		// First create the namespace
		createNamespace(t, server.App(), "contract_test_get")
		defer cleanupNamespace(server.App(), "contract_test_get")

		req := httptest.NewRequest("GET", "/v1/namespaces/contract_test_get", nil)
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

	t.Run("HeadNamespace", func(t *testing.T) {
		createNamespace(t, server.App(), "contract_test_head")
		defer cleanupNamespace(server.App(), "contract_test_head")

		req := httptest.NewRequest("HEAD", "/v1/namespaces/contract_test_head", nil)

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

		// HEAD should return 204 No Content if exists
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("UpdateNamespaceProperties", func(t *testing.T) {
		createNamespace(t, server.App(), "contract_test_update")
		defer cleanupNamespace(server.App(), "contract_test_update")

		reqBody := map[string]interface{}{
			"updates":  map[string]string{"updated_by": "test"},
			"removals": []string{},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/v1/namespaces/contract_test_update/properties", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		route, pathParams, err := spec.Router.FindRoute(req)
		require.NoError(t, err, "Route not found in spec")

		requestInput := &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		}

		req2 := httptest.NewRequest("POST", "/v1/namespaces/contract_test_update/properties", bytes.NewReader(bodyBytes))
		req2.Header.Set("Content-Type", "application/json")
		requestInput.Request = req2
		require.NoError(t, openapi3filter.ValidateRequest(context.Background(), requestInput))

		req3 := httptest.NewRequest("POST", "/v1/namespaces/contract_test_update/properties", bytes.NewReader(bodyBytes))
		req3.Header.Set("Content-Type", "application/json")
		resp, err := server.App().Test(req3, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		validateResponse(t, requestInput, resp.StatusCode, resp.Header, body)
	})

	t.Run("DeleteNamespace", func(t *testing.T) {
		createNamespace(t, server.App(), "contract_test_delete")

		req := httptest.NewRequest("DELETE", "/v1/namespaces/contract_test_delete", nil)

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

		// Should be 204 (deleted) or 404 (not found)
		assert.True(t, resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound,
			"Expected 204 or 404, got %d", resp.StatusCode)
	})
}

func testConfigWithDB() *config.Config {
	cfg := testConfig()
	cfg.Database = config.DatabaseConfig{
		Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnvOrDefault("TEST_DB_USER", "iceberg"),
		Password: getEnvOrDefault("TEST_DB_PASSWORD", "iceberg"),
		Database: getEnvOrDefault("TEST_DB_NAME", "iceberg_catalog_test"),
		SSLMode:  "disable",
	}
	return cfg
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getenv(key string) string {
	// Simple env lookup - in real code use os.Getenv
	return ""
}

func createNamespace(t *testing.T, app *fiber.App, name string) {
	t.Helper()
	reqBody := map[string]interface{}{
		"namespace":  []string{name},
		"properties": map[string]string{},
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/namespaces", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	resp.Body.Close()
}

func cleanupNamespace(app *fiber.App, name string) {
	req := httptest.NewRequest("DELETE", "/v1/namespaces/"+name, nil)
	resp, _ := app.Test(req, -1)
	if resp != nil {
		resp.Body.Close()
	}
}

func validateResponse(t *testing.T, requestInput *openapi3filter.RequestValidationInput, status int, header http.Header, body []byte) {
	t.Helper()
	responseInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: requestInput,
		Status:                 status,
		Header:                 header,
		Body:                   io.NopCloser(bytes.NewReader(body)),
	}
	err := openapi3filter.ValidateResponse(context.Background(), responseInput)
	if err != nil {
		t.Logf("Response body: %s", string(body))
	}
	assert.NoError(t, err, "Response validation failed")
}
