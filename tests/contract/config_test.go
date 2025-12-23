package contract

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/kimuyb/bingsan/internal/api"
	"github.com/kimuyb/bingsan/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfig(t *testing.T) {
	// Load OpenAPI spec
	spec, err := LoadSpec()
	require.NoError(t, err, "Failed to load OpenAPI spec")

	// Create test server
	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "GET /v1/config returns catalog configuration",
			method:         "GET",
			path:           "/v1/config",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "http://localhost"+tc.path, nil)
			req.Header.Set("Content-Type", "application/json")

			// Find route in OpenAPI spec
			route, pathParams, err := spec.Router.FindRoute(req)
			require.NoError(t, err, "Route not found in OpenAPI spec")

			// Validate request against spec
			requestInput := &openapi3filter.RequestValidationInput{
				Request:    req,
				PathParams: pathParams,
				Route:      route,
				Options: &openapi3filter.Options{
					AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
				},
			}
			err = openapi3filter.ValidateRequest(context.Background(), requestInput)
			require.NoError(t, err, "Request validation failed")

			// Execute request
			resp, err := server.App().Test(req, -1)
			require.NoError(t, err, "Request execution failed")
			defer resp.Body.Close()

			// Check status code
			assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Unexpected status code")

			// Read response body
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "Failed to read response body")

			// Validate response against spec
			responseInput := &openapi3filter.ResponseValidationInput{
				RequestValidationInput: requestInput,
				Status:                 resp.StatusCode,
				Header:                 resp.Header,
				Body:                   io.NopCloser(newBytesReader(body)),
			}
			err = openapi3filter.ValidateResponse(context.Background(), responseInput)
			assert.NoError(t, err, "Response validation failed: %s", string(body))
		})
	}
}

func testConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:    "127.0.0.1",
			Port:    0,
			Debug:   true,
			Version: "test",
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

// bytesReader wraps bytes for reading
type bytesReader struct {
	data []byte
	pos  int
}

func newBytesReader(data []byte) *bytesReader {
	return &bytesReader{data: data}
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
