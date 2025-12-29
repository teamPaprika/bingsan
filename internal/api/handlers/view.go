package handlers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kimuyb/bingsan/internal/config"
	"github.com/kimuyb/bingsan/internal/db"
	"github.com/kimuyb/bingsan/internal/pool"
)

// ViewIdentifier identifies a view by namespace and name.
type ViewIdentifier struct {
	Namespace []string `json:"namespace"`
	Name      string   `json:"name"`
}

// LoadViewResponse is the response for loading a view.
type LoadViewResponse struct {
	MetadataLocation string         `json:"metadata-location"`
	Metadata         map[string]any `json:"metadata"`
	Config           map[string]string `json:"config,omitempty"`
}

// CreateViewRequest is the request for creating a view.
type CreateViewRequest struct {
	Name       string            `json:"name"`
	Location   string            `json:"location,omitempty"`
	Schema     map[string]any    `json:"schema"`
	ViewVersion map[string]any   `json:"view-version"`
	Properties map[string]string `json:"properties,omitempty"`
}

// ReplaceViewRequest is the request for replacing a view.
type ReplaceViewRequest struct {
	Identifier   *ViewIdentifier `json:"identifier,omitempty"`
	Requirements []Requirement   `json:"requirements,omitempty"`
	Updates      []Update        `json:"updates"`
}

// RenameViewRequest is the request for renaming a view.
type RenameViewRequest struct {
	Source      ViewIdentifier `json:"source"`
	Destination ViewIdentifier `json:"destination"`
}

// ListViewsResponse is the response for listing views.
type ListViewsResponse struct {
	Identifiers []ViewIdentifier `json:"identifiers"`
	NextToken   string           `json:"next-page-token,omitempty"`
}

// ListViews returns all views in a namespace.
// GET /v1/{prefix}/namespaces/{namespace}/views
func ListViews(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))
		pageToken := c.Query("pageToken")
		pageSize := c.QueryInt("pageSize", 100)

		// Get namespace ID
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

		// List views
		query := `SELECT name FROM views WHERE namespace_id = $1 ORDER BY name`
		args := []any{namespaceID}

		if pageToken != "" {
			query = `SELECT name FROM views WHERE namespace_id = $1 AND name > $2 ORDER BY name`
			args = append(args, pageToken)
		}

		query += fmt.Sprintf(" LIMIT %d", pageSize+1)

		rows, err := database.Pool.Query(c.Context(), query, args...)
		if err != nil {
			return internalError(c, "failed to list views", err)
		}
		defer rows.Close()

		identifiers := []ViewIdentifier{}
		count := 0
		var lastName string

		for rows.Next() {
			if count >= pageSize {
				break
			}
			var name string
			if err := rows.Scan(&name); err != nil {
				return internalError(c, "failed to scan view", err)
			}
			identifiers = append(identifiers, ViewIdentifier{
				Namespace: namespaceName,
				Name:      name,
			})
			lastName = name
			count++
		}

		response := ListViewsResponse{
			Identifiers: identifiers,
		}

		if rows.Next() {
			response.NextToken = lastName
		}

		return c.JSON(response)
	}
}

// CreateView creates a new view.
// POST /v1/{prefix}/namespaces/{namespace}/views
func CreateView(database *db.DB, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))

		var req CreateViewRequest
		if err := c.BodyParser(&req); err != nil {
			return badRequest(c, "invalid request body")
		}

		if req.Name == "" {
			return badRequest(c, "view name is required")
		}

		if req.Schema == nil {
			return badRequest(c, "schema is required")
		}

		// Get namespace ID
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

		// Generate view UUID and location
		viewUUID := uuid.New().String()
		location := req.Location
		if location == "" {
			location = fmt.Sprintf("%s/%s/%s",
				cfg.Storage.Warehouse,
				strings.Join(namespaceName, "/"),
				req.Name,
			)
		}

		// Build metadata
		now := time.Now().UnixMilli()
		metadata := map[string]any{
			"format-version":      1,
			"view-uuid":           viewUUID,
			"location":            location,
			"properties":          req.Properties,
			"schemas":             []any{req.Schema},
			"current-version-id":  1,
			"versions":            []any{req.ViewVersion},
			"version-log":         []any{map[string]any{"version-id": 1, "timestamp-ms": now}},
		}

		// Generate metadata location
		metadataLocation := fmt.Sprintf("%s/metadata/00000-%.10d-%s.metadata.json",
			location, now, viewUUID[:8])

		// Use pooled buffer for JSON serialization
		buf := pool.GetBuffer()
		defer pool.PutBuffer(buf)

		encoder := json.NewEncoder(buf)
		if err := encoder.Encode(metadata); err != nil {
			return internalError(c, "failed to marshal metadata", err)
		}
		metadataJSON := buf.Bytes()

		// Insert view
		_, err = database.Pool.Exec(c.Context(), `
			INSERT INTO views (namespace_id, name, metadata_location, metadata)
			VALUES ($1, $2, $3, $4)
		`, namespaceID, req.Name, metadataLocation, metadataJSON)

		if err != nil {
			if IsDuplicateError(err) {
				return alreadyExistsError(c, "View", fmt.Sprintf("%s.%s", strings.Join(namespaceName, "."), req.Name))
			}
			return internalError(c, "failed to create view", err)
		}

		return c.JSON(LoadViewResponse{
			MetadataLocation: metadataLocation,
			Metadata:         metadata,
		})
	}
}

// LoadView retrieves view metadata.
// GET /v1/{prefix}/namespaces/{namespace}/views/{view}
func LoadView(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))
		viewName := c.Params("view")

		// Get namespace ID
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

		// Get view
		var metadataLocation string
		var metadata map[string]any
		err = database.Pool.QueryRow(c.Context(), `
			SELECT metadata_location, metadata FROM views
			WHERE namespace_id = $1 AND name = $2
		`, namespaceID, viewName).Scan(&metadataLocation, &metadata)

		if err == pgx.ErrNoRows {
			return viewNotFound(c, namespaceName, viewName)
		}
		if err != nil {
			return internalError(c, "failed to get view", err)
		}

		return c.JSON(LoadViewResponse{
			MetadataLocation: metadataLocation,
			Metadata:         metadata,
		})
	}
}

// ReplaceView replaces/updates a view.
// POST /v1/{prefix}/namespaces/{namespace}/views/{view}
func ReplaceView(database *db.DB, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))
		viewName := c.Params("view")

		var req ReplaceViewRequest
		if err := c.BodyParser(&req); err != nil {
			return badRequest(c, "invalid request body")
		}

		// Get namespace ID
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

		// Get current view state
		var viewID string
		var metadataLocation string
		var metadata map[string]any
		err = database.Pool.QueryRow(c.Context(), `
			SELECT id, metadata_location, metadata FROM views
			WHERE namespace_id = $1 AND name = $2
			FOR UPDATE
		`, namespaceID, viewName).Scan(&viewID, &metadataLocation, &metadata)

		if err == pgx.ErrNoRows {
			return viewNotFound(c, namespaceName, viewName)
		}
		if err != nil {
			return internalError(c, "failed to get view", err)
		}

		// TODO: Apply updates properly
		// For now, just update the metadata

		now := time.Now().UnixMilli()
		viewUUID, ok := getStringFromMap(metadata, "view-uuid")
		if !ok {
			return internalError(c, "invalid metadata: missing view-uuid", nil)
		}
		location, ok := getStringFromMap(metadata, "location")
		if !ok {
			return internalError(c, "invalid metadata: missing location", nil)
		}
		newMetadataLocation := fmt.Sprintf("%s/metadata/%05d-%.10d-%s.metadata.json",
			location,
			getMetadataVersion(metadataLocation)+1,
			now,
			viewUUID[:8],
		)

		// Use pooled buffer for JSON serialization
		buf := pool.GetBuffer()
		defer pool.PutBuffer(buf)

		encoder := json.NewEncoder(buf)
		if err := encoder.Encode(metadata); err != nil {
			return internalError(c, "failed to marshal metadata", err)
		}
		metadataJSON := buf.Bytes()

		// Update view
		_, err = database.Pool.Exec(c.Context(), `
			UPDATE views
			SET metadata_location = $1, metadata = $2
			WHERE id = $3
		`, newMetadataLocation, metadataJSON, viewID)

		if err != nil {
			return internalError(c, "failed to update view", err)
		}

		return c.JSON(LoadViewResponse{
			MetadataLocation: newMetadataLocation,
			Metadata:         metadata,
		})
	}
}

// DropView deletes a view.
// DELETE /v1/{prefix}/namespaces/{namespace}/views/{view}
func DropView(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))
		viewName := c.Params("view")

		// Get namespace ID
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

		// Delete view
		result, err := database.Pool.Exec(c.Context(), `
			DELETE FROM views WHERE namespace_id = $1 AND name = $2
		`, namespaceID, viewName)

		if err != nil {
			return internalError(c, "failed to delete view", err)
		}

		if result.RowsAffected() == 0 {
			return viewNotFound(c, namespaceName, viewName)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// ViewExists checks if a view exists.
// HEAD /v1/{prefix}/namespaces/{namespace}/views/{view}
func ViewExists(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))
		viewName := c.Params("view")

		var exists bool
		err := database.Pool.QueryRow(c.Context(), `
			SELECT EXISTS(
				SELECT 1 FROM views v
				JOIN namespaces n ON v.namespace_id = n.id
				WHERE n.name = $1 AND v.name = $2
			)
		`, namespaceName, viewName).Scan(&exists)

		if err != nil {
			return internalError(c, "failed to check view", err)
		}

		if !exists {
			return c.SendStatus(fiber.StatusNotFound)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// RenameView renames a view.
// POST /v1/{prefix}/views/rename
func RenameView(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req RenameViewRequest
		if err := c.BodyParser(&req); err != nil {
			return badRequest(c, "invalid request body")
		}

		// Get source namespace ID
		var sourceNamespaceID string
		err := database.Pool.QueryRow(c.Context(), `
			SELECT id FROM namespaces WHERE name = $1
		`, req.Source.Namespace).Scan(&sourceNamespaceID)

		if err == pgx.ErrNoRows {
			return namespaceNotFound(c, req.Source.Namespace)
		}
		if err != nil {
			return internalError(c, "failed to get source namespace", err)
		}

		// Get destination namespace ID
		var destNamespaceID string
		err = database.Pool.QueryRow(c.Context(), `
			SELECT id FROM namespaces WHERE name = $1
		`, req.Destination.Namespace).Scan(&destNamespaceID)

		if err == pgx.ErrNoRows {
			return namespaceNotFound(c, req.Destination.Namespace)
		}
		if err != nil {
			return internalError(c, "failed to get destination namespace", err)
		}

		// Rename view
		result, err := database.Pool.Exec(c.Context(), `
			UPDATE views
			SET namespace_id = $1, name = $2
			WHERE namespace_id = $3 AND name = $4
		`, destNamespaceID, req.Destination.Name, sourceNamespaceID, req.Source.Name)

		if err != nil {
			if IsDuplicateError(err) {
				return alreadyExistsError(c, "View", fmt.Sprintf("%s.%s",
					strings.Join(req.Destination.Namespace, "."), req.Destination.Name))
			}
			return internalError(c, "failed to rename view", err)
		}

		if result.RowsAffected() == 0 {
			return viewNotFound(c, req.Source.Namespace, req.Source.Name)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// viewNotFound returns a view not found error.
func viewNotFound(c *fiber.Ctx, namespace []string, view string) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": fiber.Map{
			"message": fmt.Sprintf("View does not exist: %s.%s", strings.Join(namespace, "."), view),
			"type":    "NoSuchViewException",
			"code":    404,
		},
	})
}
