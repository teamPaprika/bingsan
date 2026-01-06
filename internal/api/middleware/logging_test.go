package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	app := fiber.New()
	app.Use(Logger())

	t.Run("successful request", func(t *testing.T) {
		app.Get("/success", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/success", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("client error", func(t *testing.T) {
		app.Get("/client-error", func(c *fiber.Ctx) error {
			return c.Status(400).SendString("Bad Request")
		})

		req := httptest.NewRequest("GET", "/client-error", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("server error", func(t *testing.T) {
		app.Get("/server-error", func(c *fiber.Ctx) error {
			return c.Status(500).SendString("Internal Server Error")
		})

		req := httptest.NewRequest("GET", "/server-error", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 500, resp.StatusCode)
	})

	t.Run("with error", func(t *testing.T) {
		app.Get("/with-error", func(c *fiber.Ctx) error {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		})

		req := httptest.NewRequest("GET", "/with-error", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})
}

func TestLogger_Methods(t *testing.T) {
	app := fiber.New()
	app.Use(Logger())

	handler := func(c *fiber.Ctx) error {
		return c.SendString("OK")
	}

	app.Get("/test", handler)
	app.Post("/test", handler)
	app.Put("/test", handler)
	app.Delete("/test", handler)

	methods := []string{"GET", "POST", "PUT", "DELETE"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)
		})
	}
}

func TestLogger_StatusCodes(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"200 OK", 200},
		{"201 Created", 201},
		{"204 No Content", 204},
		{"301 Moved Permanently", 301},
		{"400 Bad Request", 400},
		{"401 Unauthorized", 401},
		{"403 Forbidden", 403},
		{"404 Not Found", 404},
		{"500 Internal Server Error", 500},
		{"502 Bad Gateway", 502},
		{"503 Service Unavailable", 503},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(Logger())
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendStatus(tt.status)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.status, resp.StatusCode)
		})
	}
}

func TestLogger_ReturnsError(t *testing.T) {
	app := fiber.New()
	app.Use(Logger())
	app.Get("/error", func(c *fiber.Ctx) error {
		return fiber.ErrInternalServerError
	})

	req := httptest.NewRequest("GET", "/error", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
}
