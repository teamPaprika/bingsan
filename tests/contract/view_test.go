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
	"github.com/teamPaprika/bingsan/internal/api"
	"github.com/teamPaprika/bingsan/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestViewOperations tests view CRUD operations against the OpenAPI spec.
func TestViewOperations(t *testing.T) {
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
	createNamespace(t, server.App(), "view_test_ns")
	defer cleanupNamespace(server.App(), "view_test_ns")

	t.Run("ListViews", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/namespaces/view_test_ns/views", nil)
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

	t.Run("CreateView", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "contract_test_view",
			"schema": map[string]interface{}{
				"schema-id": 0,
				"type":      "struct",
				"fields": []map[string]interface{}{
					{"id": 1, "name": "id", "type": "long", "required": true},
				},
			},
			"view-version": map[string]interface{}{
				"version-id": 1,
				"schema-id":  0,
				"representations": []map[string]interface{}{
					{
						"type":    "sql",
						"sql":     "SELECT * FROM test_table",
						"dialect": "spark",
					},
				},
			},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/v1/namespaces/view_test_ns/views", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		route, pathParams, err := spec.Router.FindRoute(req)
		require.NoError(t, err, "Route not found in spec")

		requestInput := &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		}

		req2 := httptest.NewRequest("POST", "/v1/namespaces/view_test_ns/views", bytes.NewReader(bodyBytes))
		req2.Header.Set("Content-Type", "application/json")
		requestInput.Request = req2
		err = openapi3filter.ValidateRequest(context.Background(), requestInput)
		require.NoError(t, err, "Request validation failed")

		req3 := httptest.NewRequest("POST", "/v1/namespaces/view_test_ns/views", bytes.NewReader(bodyBytes))
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

		cleanupView(server.App(), "view_test_ns", "contract_test_view")
	})

	t.Run("LoadView", func(t *testing.T) {
		createView(t, server.App(), "view_test_ns", "load_test_view")
		defer cleanupView(server.App(), "view_test_ns", "load_test_view")

		req := httptest.NewRequest("GET", "/v1/namespaces/view_test_ns/views/load_test_view", nil)
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

	t.Run("HeadView", func(t *testing.T) {
		createView(t, server.App(), "view_test_ns", "head_test_view")
		defer cleanupView(server.App(), "view_test_ns", "head_test_view")

		req := httptest.NewRequest("HEAD", "/v1/namespaces/view_test_ns/views/head_test_view", nil)

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

	t.Run("DropView", func(t *testing.T) {
		createView(t, server.App(), "view_test_ns", "drop_test_view")

		req := httptest.NewRequest("DELETE", "/v1/namespaces/view_test_ns/views/drop_test_view", nil)

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

func createView(t *testing.T, app *fiber.App, namespace, viewName string) {
	t.Helper()
	reqBody := map[string]interface{}{
		"name": viewName,
		"schema": map[string]interface{}{
			"schema-id": 0,
			"type":      "struct",
			"fields": []map[string]interface{}{
				{"id": 1, "name": "id", "type": "long", "required": true},
			},
		},
		"view-version": map[string]interface{}{
			"version-id": 1,
			"schema-id":  0,
			"representations": []map[string]interface{}{
				{
					"type":    "sql",
					"sql":     "SELECT 1",
					"dialect": "spark",
				},
			},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/namespaces/"+namespace+"/views", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp != nil {
		resp.Body.Close()
	}
}

func cleanupView(app *fiber.App, namespace, viewName string) {
	req := httptest.NewRequest("DELETE", "/v1/namespaces/"+namespace+"/views/"+viewName, nil)
	resp, _ := app.Test(req, -1)
	if resp != nil {
		resp.Body.Close()
	}
}
