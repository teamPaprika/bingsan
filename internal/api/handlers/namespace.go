package handlers

import (
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"

	"github.com/kimuyb/bingsan/internal/db"
	"github.com/kimuyb/bingsan/internal/events"
	"github.com/kimuyb/bingsan/internal/metrics"
)

// Namespace represents a namespace in the catalog.
type Namespace struct {
	Namespace  []string          `json:"namespace"`
	Properties map[string]string `json:"properties,omitempty"`
}

// CreateNamespaceRequest is the request body for creating a namespace.
type CreateNamespaceRequest struct {
	Namespace  []string          `json:"namespace"`
	Properties map[string]string `json:"properties,omitempty"`
}

// ListNamespacesResponse is the response for listing namespaces.
type ListNamespacesResponse struct {
	Namespaces [][]string `json:"namespaces"`
	NextToken  string     `json:"next-page-token,omitempty"`
}

// UpdateNamespacePropertiesRequest is the request for updating namespace properties.
type UpdateNamespacePropertiesRequest struct {
	Removals []string          `json:"removals,omitempty"`
	Updates  map[string]string `json:"updates,omitempty"`
}

// UpdateNamespacePropertiesResponse is the response for updating namespace properties.
type UpdateNamespacePropertiesResponse struct {
	Updated  []string `json:"updated"`
	Removed  []string `json:"removed"`
	Missing  []string `json:"missing,omitempty"`
}

// ListNamespaces returns all namespaces.
// GET /v1/{prefix}/namespaces
func ListNamespaces(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Parse query parameters
		parent := c.Query("parent")
		pageToken := c.Query("pageToken")
		pageSize := validatePageSize(c.QueryInt("pageSize", 100))

		// Build query with proper parameterized arguments
		var query string
		var args []any

		if parent != "" {
			parentParts := strings.Split(parent, "\x1f")
			if pageToken != "" {
				query = `SELECT name FROM namespaces WHERE name[1:array_length($1::text[], 1)] = $1 AND name > $2 ORDER BY name LIMIT $3`
				args = []any{parentParts, strings.Split(pageToken, "\x1f"), pageSize + 1}
			} else {
				query = `SELECT name FROM namespaces WHERE name[1:array_length($1::text[], 1)] = $1 ORDER BY name LIMIT $2`
				args = []any{parentParts, pageSize + 1}
			}
		} else {
			if pageToken != "" {
				query = `SELECT name FROM namespaces WHERE name > $1 ORDER BY name LIMIT $2`
				args = []any{strings.Split(pageToken, "\x1f"), pageSize + 1}
			} else {
				query = `SELECT name FROM namespaces ORDER BY name LIMIT $1`
				args = []any{pageSize + 1}
			}
		}

		rows, err := database.Pool.Query(c.Context(), query, args...)
		if err != nil {
			return internalError(c, "failed to list namespaces", err)
		}
		defer rows.Close()

		// Preallocate slice with expected capacity
		namespaces := make([][]string, 0, pageSize)
		var lastName []string
		count := 0
		for rows.Next() {
			if count >= pageSize {
				break // We have more, will set next page token
			}
			var name []string
			if err := rows.Scan(&name); err != nil {
				return internalError(c, "failed to scan namespace", err)
			}
			namespaces = append(namespaces, name)
			lastName = name
			count++
		}

		response := ListNamespacesResponse{
			Namespaces: namespaces,
		}

		// Set next page token using the last namespace name as cursor
		if rows.Next() && len(lastName) > 0 {
			response.NextToken = strings.Join(lastName, "\x1f")
		}

		return c.JSON(response)
	}
}

// CreateNamespace creates a new namespace.
// POST /v1/{prefix}/namespaces
func CreateNamespace(database *db.DB, auditLogger *events.AuditLogger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req CreateNamespaceRequest
		if err := c.BodyParser(&req); err != nil {
			return badRequest(c, "invalid request body")
		}

		if len(req.Namespace) == 0 {
			return badRequest(c, "namespace is required")
		}

		// Convert properties to JSONB
		properties := req.Properties
		if properties == nil {
			properties = map[string]string{}
		}

		// Insert namespace
		_, err := database.Pool.Exec(c.Context(), `
			INSERT INTO namespaces (name, properties)
			VALUES ($1, $2)
		`, req.Namespace, properties)

		if err != nil {
			if IsDuplicateError(err) {
				return alreadyExistsError(c, "Namespace", strings.Join(req.Namespace, "."))
			}
			return internalError(c, "failed to create namespace", err)
		}

		// Record metrics
		metrics.RecordNamespaceCreated()

		// Log audit event
		event := events.NewEvent(events.NamespaceCreated).
			WithNamespace(strings.Join(req.Namespace, "."))
		LogAudit(c, auditLogger, event, fiber.StatusOK)

		return c.Status(fiber.StatusOK).JSON(Namespace{
			Namespace:  req.Namespace,
			Properties: properties,
		})
	}
}

// GetNamespace retrieves namespace metadata.
// GET /v1/{prefix}/namespaces/{namespace}
func GetNamespace(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))

		var properties map[string]string
		err := database.Pool.QueryRow(c.Context(), `
			SELECT properties FROM namespaces WHERE name = $1
		`, namespaceName).Scan(&properties)

		if err == pgx.ErrNoRows {
			return namespaceNotFound(c, namespaceName)
		}
		if err != nil {
			return internalError(c, "failed to get namespace", err)
		}

		return c.JSON(Namespace{
			Namespace:  namespaceName,
			Properties: properties,
		})
	}
}

// NamespaceExists checks if a namespace exists.
// HEAD /v1/{prefix}/namespaces/{namespace}
func NamespaceExists(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))

		var exists bool
		err := database.Pool.QueryRow(c.Context(), `
			SELECT EXISTS(SELECT 1 FROM namespaces WHERE name = $1)
		`, namespaceName).Scan(&exists)

		if err != nil {
			return internalError(c, "failed to check namespace", err)
		}

		if !exists {
			return c.SendStatus(fiber.StatusNotFound)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// DeleteNamespace deletes an empty namespace.
// DELETE /v1/{prefix}/namespaces/{namespace}
func DeleteNamespace(database *db.DB, auditLogger *events.AuditLogger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))

		// Check if namespace exists
		var namespaceID string
		err := database.Pool.QueryRow(c.Context(), `
			SELECT id FROM namespaces WHERE name = $1
		`, namespaceName).Scan(&namespaceID)

		if err == pgx.ErrNoRows {
			return namespaceNotFound(c, namespaceName)
		}
		if err != nil {
			return internalError(c, "failed to get namespace", err)
		}

		// Check if namespace has any tables or views
		var count int
		err = database.Pool.QueryRow(c.Context(), `
			SELECT COUNT(*) FROM (
				SELECT 1 FROM tables WHERE namespace_id = $1
				UNION ALL
				SELECT 1 FROM views WHERE namespace_id = $1
			) AS items
		`, namespaceID).Scan(&count)

		if err != nil {
			return internalError(c, "failed to check namespace contents", err)
		}

		if count > 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "Namespace is not empty",
					"type":    "BadRequestException",
					"code":    400,
				},
			})
		}

		// Delete namespace
		_, err = database.Pool.Exec(c.Context(), `
			DELETE FROM namespaces WHERE id = $1
		`, namespaceID)

		if err != nil {
			return internalError(c, "failed to delete namespace", err)
		}

		// Record metrics
		metrics.RecordNamespaceDropped()

		// Log audit event
		event := events.NewEvent(events.NamespaceDropped).
			WithNamespace(strings.Join(namespaceName, "."))
		LogAudit(c, auditLogger, event, fiber.StatusNoContent)

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// UpdateNamespaceProperties updates namespace properties.
// POST /v1/{prefix}/namespaces/{namespace}/properties
func UpdateNamespaceProperties(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))

		var req UpdateNamespacePropertiesRequest
		if err := c.BodyParser(&req); err != nil {
			return badRequest(c, "invalid request body")
		}

		// Get current properties
		var properties map[string]string
		err := database.Pool.QueryRow(c.Context(), `
			SELECT properties FROM namespaces WHERE name = $1
		`, namespaceName).Scan(&properties)

		if err == pgx.ErrNoRows {
			return namespaceNotFound(c, namespaceName)
		}
		if err != nil {
			return internalError(c, "failed to get namespace", err)
		}

		if properties == nil {
			properties = map[string]string{}
		}

		// Track changes
		updated := []string{}
		removed := []string{}
		missing := []string{}

		// Apply updates
		for key, value := range req.Updates {
			properties[key] = value
			updated = append(updated, key)
		}

		// Apply removals
		for _, key := range req.Removals {
			if _, exists := properties[key]; exists {
				delete(properties, key)
				removed = append(removed, key)
			} else {
				missing = append(missing, key)
			}
		}

		// Save updated properties
		_, err = database.Pool.Exec(c.Context(), `
			UPDATE namespaces SET properties = $1 WHERE name = $2
		`, properties, namespaceName)

		if err != nil {
			return internalError(c, "failed to update namespace", err)
		}

		return c.JSON(UpdateNamespacePropertiesResponse{
			Updated: updated,
			Removed: removed,
			Missing: missing,
		})
	}
}

// parseNamespace converts URL-encoded namespace to parts.
func parseNamespace(encoded string) []string {
	// URL-decode the namespace first (Fiber doesn't auto-decode path params)
	decoded, err := url.PathUnescape(encoded)
	if err != nil {
		// Fallback to original if decode fails
		decoded = encoded
	}
	// URL encoding uses %1F as separator, which decodes to \x1f
	// Also handle dot-separated for simple cases
	if strings.Contains(decoded, "\x1f") {
		return strings.Split(decoded, "\x1f")
	}
	return strings.Split(decoded, ".")
}

// namespaceNotFound returns a namespace not found error.
func namespaceNotFound(c *fiber.Ctx, namespace []string) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": fiber.Map{
			"message": "Namespace does not exist: " + strings.Join(namespace, "."),
			"type":    "NoSuchNamespaceException",
			"code":    404,
		},
	})
}

// badRequest returns a bad request error.
func badRequest(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": fiber.Map{
			"message": message,
			"type":    "BadRequestException",
			"code":    400,
		},
	})
}

// internalError returns an internal server error.
func internalError(c *fiber.Ctx, message string, err error) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": fiber.Map{
			"message": message + ": " + err.Error(),
			"type":    "ServerError",
			"code":    500,
		},
	})
}
