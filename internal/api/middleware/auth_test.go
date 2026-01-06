package middleware

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kimuyb/bingsan/internal/config"
)

func TestContextKey(t *testing.T) {
	assert.Equal(t, ContextKey("principal"), ContextKeyPrincipal)
}

func TestPrincipal(t *testing.T) {
	p := Principal{
		ID:       "user-123",
		Type:     "api_key",
		Scopes:   []string{"read", "write"},
		ClientID: "client-456",
	}

	assert.Equal(t, "user-123", p.ID)
	assert.Equal(t, "api_key", p.Type)
	assert.Equal(t, []string{"read", "write"}, p.Scopes)
	assert.Equal(t, "client-456", p.ClientID)
}

func TestBuildSkipPaths(t *testing.T) {
	t.Run("polaris disabled", func(t *testing.T) {
		paths := buildSkipPaths(false)

		assert.Contains(t, paths, "/health")
		assert.Contains(t, paths, "/ready")
		assert.Contains(t, paths, "/v1/config")
		assert.Contains(t, paths, "/v1/oauth/tokens")
		assert.NotContains(t, paths, "/api/catalog/v1/oauth/tokens")
	})

	t.Run("polaris enabled", func(t *testing.T) {
		paths := buildSkipPaths(true)

		assert.Contains(t, paths, "/health")
		assert.Contains(t, paths, "/ready")
		assert.Contains(t, paths, "/v1/config")
		assert.Contains(t, paths, "/v1/oauth/tokens")
		assert.Contains(t, paths, "/api/catalog/v1/oauth/tokens")
	})
}

func TestShouldSkipAuth(t *testing.T) {
	skipPaths := []string{"/health", "/ready", "/v1/config"}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"health endpoint", "/health", true},
		{"ready endpoint", "/ready", true},
		{"config endpoint", "/v1/config", true},
		{"api endpoint", "/v1/namespaces", false},
		{"nested endpoint", "/v1/namespaces/test/tables", false},
		{"partial match", "/health/live", false},
		{"empty path", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSkipAuth(tt.path, skipPaths)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"simple token", "my-secret-token"},
		{"empty token", ""},
		{"special characters", "token!@#$%^&*()"},
		{"long token", "a-very-long-token-that-should-still-work-properly-even-with-many-characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := hashToken(tt.token)

			// SHA-256 produces 64 hex characters
			assert.Len(t, hash, 64)

			// Same input should produce same hash
			assert.Equal(t, hash, hashToken(tt.token))

			// Different inputs should produce different hashes
			if tt.token != "" {
				assert.NotEqual(t, hash, hashToken(tt.token+"x"))
			}
		})
	}
}

func TestUnauthorizedError(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return unauthorizedError(c, "test error message")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, 401, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "test error message")
	assert.Contains(t, string(body), "NotAuthorizedException")
	assert.Contains(t, string(body), "401")
}

func TestGetPrincipal(t *testing.T) {
	app := fiber.New()

	t.Run("with principal", func(t *testing.T) {
		principal := &Principal{
			ID:       "user-123",
			Type:     "api_key",
			Scopes:   []string{"read"},
			ClientID: "client-456",
		}

		app.Get("/with-principal", func(c *fiber.Ctx) error {
			c.Locals(string(ContextKeyPrincipal), principal)
			retrieved := GetPrincipal(c)
			if retrieved == nil {
				return c.SendStatus(500)
			}
			return c.JSON(retrieved)
		})

		req := httptest.NewRequest("GET", "/with-principal", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("without principal", func(t *testing.T) {
		app.Get("/without-principal", func(c *fiber.Ctx) error {
			retrieved := GetPrincipal(c)
			if retrieved != nil {
				return c.SendStatus(500)
			}
			return c.SendStatus(200)
		})

		req := httptest.NewRequest("GET", "/without-principal", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("wrong type in locals", func(t *testing.T) {
		app.Get("/wrong-type", func(c *fiber.Ctx) error {
			c.Locals(string(ContextKeyPrincipal), "not a principal")
			retrieved := GetPrincipal(c)
			if retrieved != nil {
				return c.SendStatus(500)
			}
			return c.SendStatus(200)
		})

		req := httptest.NewRequest("GET", "/wrong-type", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

func TestAuth_SkipPaths(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			Enabled: true,
		},
		Compat: config.CompatConfig{
			PolarisEnabled: false,
		},
	}

	app := fiber.New()
	app.Use(Auth(cfg, nil))
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Get("/ready", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Get("/v1/config", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	t.Run("health skips auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("ready skips auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ready", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("config skips auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/config", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

func TestAuth_MissingHeader(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			Enabled: true,
		},
	}

	app := fiber.New()
	app.Use(Auth(cfg, nil))
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "missing authorization header")
}

func TestAuth_InvalidHeaderFormat(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			Enabled: true,
		},
	}

	app := fiber.New()
	app.Use(Auth(cfg, nil))
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "invalid authorization header format")
}

func TestAuth_UnsupportedScheme(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			Enabled: true,
		},
	}

	app := fiber.New()
	app.Use(Auth(cfg, nil))
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "unsupported authentication scheme")
}

func TestAuth_BearerNotEnabled(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			Enabled: true,
			OAuth2: config.OAuth2Config{
				Enabled: false,
			},
			APIKey: config.APIKeyConfig{
				Enabled: false,
			},
		},
	}

	app := fiber.New()
	app.Use(Auth(cfg, nil))
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "bearer authentication not enabled")
}

func TestAuth_APIKeyNotEnabled(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			Enabled: true,
			APIKey: config.APIKeyConfig{
				Enabled: false,
			},
		},
	}

	app := fiber.New()
	app.Use(Auth(cfg, nil))
	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "X-API-Key some-api-key")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "API key authentication not enabled")
}

func TestRotationGracePeriod(t *testing.T) {
	assert.Equal(t, 24*3600*1000*1000*1000, int(RotationGracePeriod.Nanoseconds()))
}
