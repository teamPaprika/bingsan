package handlers

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck(t *testing.T) {
	app := fiber.New()
	app.Get("/health", HealthCheck)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, 200, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "healthy")
	assert.Contains(t, string(body), "status")
}

func TestReadyCheck_NilDatabase(t *testing.T) {
	app := fiber.New()
	app.Get("/ready", ReadyCheck(nil))

	req := httptest.NewRequest("GET", "/ready", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, 503, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "not ready")
	assert.Contains(t, string(body), "not configured")
}
