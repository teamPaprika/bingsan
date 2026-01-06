package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/kimuyb/bingsan/internal/api/middleware"
	"github.com/kimuyb/bingsan/internal/events"
)

// AuditFromContext creates an AuditEvent from a CatalogEvent, enriched with
// request context from the Fiber context.
func AuditFromContext(c *fiber.Ctx, event *events.CatalogEvent) *events.AuditEvent {
	audit := events.NewAuditEvent(event)

	// Get authenticated principal for actor info
	if principal := middleware.GetPrincipal(c); principal != nil {
		audit.CatalogEvent.Actor = principal.ClientID
		audit.ActorType = principal.Type
	}

	// Add request context
	audit.IPAddress = c.IP()
	audit.UserAgent = c.Get("User-Agent")

	// Generate or get request ID
	requestID := c.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
	}
	audit.RequestID = requestID

	return audit
}

// LogAudit logs an audit event with response code.
func LogAudit(c *fiber.Ctx, auditLogger *events.AuditLogger, event *events.CatalogEvent, statusCode int) {
	if auditLogger == nil {
		return
	}

	audit := AuditFromContext(c, event)
	audit.ResponseCode = statusCode

	// Fire and forget - don't block on audit logging
	go func() {
		_ = auditLogger.Log(c.Context(), audit)
	}()
}
