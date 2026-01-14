package middleware

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/teamPaprika/bingsan/internal/config"
	"github.com/teamPaprika/bingsan/internal/db"
)

// Permission represents an access permission level.
type Permission string

const (
	// PermissionRead allows read-only access to namespace resources.
	PermissionRead Permission = "read"
	// PermissionWrite allows read and write access to namespace resources.
	PermissionWrite Permission = "write"
	// PermissionManage allows full control including namespace deletion.
	PermissionManage Permission = "manage"
)

// Authz returns an authorization middleware that checks namespace permissions.
func Authz(cfg *config.Config, database *db.DB, required Permission) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip if ACL is disabled
		if !cfg.Auth.ACL.Enabled {
			return c.Next()
		}

		principal := GetPrincipal(c)
		if principal == nil {
			return forbiddenError(c, "no authenticated principal")
		}

		// Extract namespace from path
		namespace := extractNamespace(c.Path())
		if namespace == "" {
			// Non-namespaced routes pass through
			return c.Next()
		}

		// Check permission in database
		hasPermission, err := checkPermission(c.Context(), database, principal, namespace, required)
		if err != nil {
			return forbiddenError(c, "permission check failed")
		}

		if !hasPermission {
			// Check default permission as fallback
			if hasDefaultPermission(cfg.Auth.ACL.DefaultPermission, required) {
				return c.Next()
			}
			return forbiddenError(c, "insufficient permissions for namespace")
		}

		return c.Next()
	}
}

// extractNamespace extracts the namespace from the request path.
// Handles paths like /v1/namespaces/{namespace}/... and /api/catalog/v1/namespaces/{namespace}/...
func extractNamespace(path string) string {
	parts := strings.Split(path, "/")

	// Find "namespaces" in the path and get the next segment
	for i, part := range parts {
		if part == "namespaces" && i+1 < len(parts) {
			ns := parts[i+1]
			// Namespace can contain encoded dots (e.g., "db%2Eschema")
			// but for now return as-is; decoding happens at handler level
			if ns != "" && ns != "tables" && ns != "views" && ns != "properties" {
				return ns
			}
		}
	}

	return ""
}

// checkPermission checks if a principal has the required permission on a namespace.
func checkPermission(ctx context.Context, database *db.DB, principal *Principal, namespace string, required Permission) (bool, error) {
	var hasPermission bool

	// Query the permission function we created in the migration
	err := database.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM namespace_permissions np
			JOIN namespaces n ON n.id = np.namespace_id
			WHERE n.name[1] = $1
			  AND np.principal_id = $2
			  AND np.principal_type = $3
			  AND $4 = ANY(np.permissions)
		)
	`, namespace, principal.ID, principal.Type, string(required)).Scan(&hasPermission)

	if err != nil {
		return false, err
	}

	// If checking for read/write, also accept manage permission
	if !hasPermission && required != PermissionManage {
		err = database.Pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM namespace_permissions np
				JOIN namespaces n ON n.id = np.namespace_id
				WHERE n.name[1] = $1
				  AND np.principal_id = $2
				  AND np.principal_type = $3
				  AND 'manage' = ANY(np.permissions)
			)
		`, namespace, principal.ID, principal.Type).Scan(&hasPermission)
		if err != nil {
			return false, err
		}
	}

	// If checking for read, also accept write permission
	if !hasPermission && required == PermissionRead {
		err = database.Pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM namespace_permissions np
				JOIN namespaces n ON n.id = np.namespace_id
				WHERE n.name[1] = $1
				  AND np.principal_id = $2
				  AND np.principal_type = $3
				  AND 'write' = ANY(np.permissions)
			)
		`, namespace, principal.ID, principal.Type).Scan(&hasPermission)
		if err != nil {
			return false, err
		}
	}

	return hasPermission, nil
}

// hasDefaultPermission checks if the default permission satisfies the required permission.
func hasDefaultPermission(defaultPerm string, required Permission) bool {
	switch defaultPerm {
	case "manage":
		return true
	case "write":
		return required == PermissionRead || required == PermissionWrite
	case "read":
		return required == PermissionRead
	default:
		return false
	}
}

// forbiddenError returns a standardized forbidden error.
func forbiddenError(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
		"error": fiber.Map{
			"message": message,
			"type":    "ForbiddenException",
			"code":    403,
		},
	})
}
