package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kimuyb/bingsan/internal/events"
)

func TestAuditFromContext(t *testing.T) {
	t.Run("creates audit event with IP and user agent", func(t *testing.T) {
		app := fiber.New()
		var capturedAudit *events.AuditEvent

		app.Get("/test", func(c *fiber.Ctx) error {
			event := &events.CatalogEvent{
				Type:      events.TableCreated,
				Namespace: "ns",
				Table:     "tbl",
			}
			capturedAudit = AuditFromContext(c, event)
			return c.SendStatus(200)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "TestAgent/1.0")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		require.NotNil(t, capturedAudit)
		assert.Equal(t, events.TableCreated, capturedAudit.Type)
		assert.Equal(t, "ns", capturedAudit.Namespace)
		assert.Equal(t, "tbl", capturedAudit.Table)
		assert.Equal(t, "TestAgent/1.0", capturedAudit.UserAgent)
		assert.NotEmpty(t, capturedAudit.IPAddress)
	})

	t.Run("uses provided X-Request-ID", func(t *testing.T) {
		app := fiber.New()
		var capturedAudit *events.AuditEvent

		app.Get("/test", func(c *fiber.Ctx) error {
			event := &events.CatalogEvent{Type: events.TableCreated}
			capturedAudit = AuditFromContext(c, event)
			return c.SendStatus(200)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", "custom-request-id")

		_, err := app.Test(req)
		require.NoError(t, err)

		require.NotNil(t, capturedAudit)
		assert.Equal(t, "custom-request-id", capturedAudit.RequestID)
	})

	t.Run("generates request ID when not provided", func(t *testing.T) {
		app := fiber.New()
		var capturedAudit *events.AuditEvent

		app.Get("/test", func(c *fiber.Ctx) error {
			event := &events.CatalogEvent{Type: events.TableCreated}
			capturedAudit = AuditFromContext(c, event)
			return c.SendStatus(200)
		})

		req := httptest.NewRequest("GET", "/test", nil)

		_, err := app.Test(req)
		require.NoError(t, err)

		require.NotNil(t, capturedAudit)
		assert.NotEmpty(t, capturedAudit.RequestID)
		// Should be a UUID format
		assert.Len(t, capturedAudit.RequestID, 36)
	})
}

func TestLogAudit_NilLogger(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		event := &events.CatalogEvent{
			Type:      events.TableCreated,
			Namespace: "ns",
			Table:     "tbl",
		}
		// Should not panic with nil logger
		LogAudit(c, nil, event, 200)
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
