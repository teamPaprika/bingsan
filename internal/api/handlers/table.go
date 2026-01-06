package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kimuyb/bingsan/internal/config"
	"github.com/kimuyb/bingsan/internal/db"
	"github.com/kimuyb/bingsan/internal/events"
	"github.com/kimuyb/bingsan/internal/metrics"
	"github.com/kimuyb/bingsan/internal/pool"
)

// TableIdentifier identifies a table by namespace and name.
type TableIdentifier struct {
	Namespace []string `json:"namespace"`
	Name      string   `json:"name"`
}

// TableMetadata represents Iceberg table metadata.
type TableMetadata struct {
	FormatVersion      int                    `json:"format-version"`
	TableUUID          string                 `json:"table-uuid"`
	Location           string                 `json:"location"`
	LastUpdatedMs      int64                  `json:"last-updated-ms"`
	Properties         map[string]string      `json:"properties,omitempty"`
	Schemas            []map[string]any       `json:"schemas,omitempty"`
	CurrentSchemaID    int                    `json:"current-schema-id"`
	PartitionSpecs     []map[string]any       `json:"partition-specs,omitempty"`
	DefaultSpecID      int                    `json:"default-spec-id"`
	SortOrders         []map[string]any       `json:"sort-orders,omitempty"`
	DefaultSortOrderID int                    `json:"default-sort-order-id"`
	Snapshots          []map[string]any       `json:"snapshots,omitempty"`
	SnapshotLog        []map[string]any       `json:"snapshot-log,omitempty"`
	Refs               map[string]any         `json:"refs,omitempty"`
	CurrentSnapshotID  *int64                 `json:"current-snapshot-id,omitempty"`
	LastSequenceNumber int64                  `json:"last-sequence-number,omitempty"`
	LastColumnID       int                    `json:"last-column-id,omitempty"`
	NextRowID          *int64                 `json:"next-row-id,omitempty"`
	Statistics         []map[string]any       `json:"statistics,omitempty"`
	PartitionStatistics []map[string]any      `json:"partition-statistics,omitempty"`
}

// LoadTableResponse is the response for loading a table.
type LoadTableResponse struct {
	MetadataLocation string                 `json:"metadata-location"`
	Metadata         map[string]any         `json:"metadata"`
	Config           map[string]string      `json:"config,omitempty"`
}

// CreateTableRequest is the request for creating a table.
type CreateTableRequest struct {
	Name           string                 `json:"name"`
	Location       string                 `json:"location,omitempty"`
	Schema         map[string]any         `json:"schema"`
	PartitionSpec  map[string]any         `json:"partition-spec,omitempty"`
	SortOrder      map[string]any         `json:"write-order,omitempty"`
	StageCreate    bool                   `json:"stage-create,omitempty"`
	Properties     map[string]string      `json:"properties,omitempty"`
}

// CommitTableRequest is the request for committing table changes.
type CommitTableRequest struct {
	Identifier   *TableIdentifier `json:"identifier,omitempty"`
	Requirements []Requirement    `json:"requirements"`
	Updates      []Update         `json:"updates"`
}

// Requirement is a commit requirement.
type Requirement struct {
	Type                  string `json:"type"`
	Ref                   string `json:"ref,omitempty"`
	UUID                  string `json:"uuid,omitempty"`
	SnapshotID            *int64 `json:"snapshot-id,omitempty"`
	LastAssignedFieldID   *int   `json:"last-assigned-field-id,omitempty"`
	CurrentSchemaID       *int   `json:"current-schema-id,omitempty"`
	LastAssignedPartitionID *int `json:"last-assigned-partition-id,omitempty"`
	DefaultSpecID         *int   `json:"default-spec-id,omitempty"`
	DefaultSortOrderID    *int   `json:"default-sort-order-id,omitempty"`
}

// Update is a table update operation.
type Update struct {
	Action string         `json:"action"`
	// Different actions have different fields
	// We use json.RawMessage to handle this dynamically
}

// RenameTableRequest is the request for renaming a table.
type RenameTableRequest struct {
	Source      TableIdentifier `json:"source"`
	Destination TableIdentifier `json:"destination"`
}

// RegisterTableRequest is the request for registering a table.
type RegisterTableRequest struct {
	Name             string `json:"name"`
	MetadataLocation string `json:"metadata-location"`
}

// ListTablesResponse is the response for listing tables.
type ListTablesResponse struct {
	Identifiers []TableIdentifier `json:"identifiers"`
	NextToken   string            `json:"next-page-token,omitempty"`
}

// ListTables returns all tables in a namespace.
// GET /v1/{prefix}/namespaces/{namespace}/tables
func ListTables(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))
		pageToken := c.Query("pageToken")
		pageSize := validatePageSize(c.QueryInt("pageSize", 100))

		// Single query with JOIN - eliminates N+1
		var query string
		var args []any

		if pageToken != "" {
			query = `
				SELECT t.name FROM tables t
				JOIN namespaces n ON t.namespace_id = n.id
				WHERE n.name = $1 AND t.name > $2
				ORDER BY t.name LIMIT $3`
			args = []any{namespaceName, pageToken, pageSize + 1}
		} else {
			query = `
				SELECT t.name FROM tables t
				JOIN namespaces n ON t.namespace_id = n.id
				WHERE n.name = $1
				ORDER BY t.name LIMIT $2`
			args = []any{namespaceName, pageSize + 1}
		}

		rows, err := database.Pool.Query(c.Context(), query, args...)
		if err != nil {
			return internalError(c, "failed to list tables", err)
		}
		defer rows.Close()

		// Preallocate slice with expected capacity
		identifiers := make([]TableIdentifier, 0, pageSize)
		count := 0
		var lastName string

		for rows.Next() {
			if count >= pageSize {
				break
			}
			var name string
			if err := rows.Scan(&name); err != nil {
				return internalError(c, "failed to scan table", err)
			}
			identifiers = append(identifiers, TableIdentifier{
				Namespace: namespaceName,
				Name:      name,
			})
			lastName = name
			count++
		}

		// Check if namespace exists (empty result could mean no tables OR no namespace)
		if count == 0 {
			var exists bool
			err := database.Pool.QueryRow(c.Context(), `
				SELECT EXISTS(SELECT 1 FROM namespaces WHERE name = $1)
			`, namespaceName).Scan(&exists)
			if err != nil {
				return internalError(c, "failed to check namespace", err)
			}
			if !exists {
				return namespaceNotFound(c, namespaceName)
			}
		}

		response := ListTablesResponse{
			Identifiers: identifiers,
		}

		if rows.Next() {
			response.NextToken = lastName
		}

		return c.JSON(response)
	}
}

// CreateTable creates a new table.
// POST /v1/{prefix}/namespaces/{namespace}/tables
func CreateTable(database *db.DB, cfg *config.Config, auditLogger *events.AuditLogger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))

		var req CreateTableRequest
		if err := c.BodyParser(&req); err != nil {
			return badRequest(c, "invalid request body")
		}

		if req.Name == "" {
			return badRequest(c, "table name is required")
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

		// Generate table UUID and location
		tableUUID := uuid.New().String()
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
			"format-version":    2,
			"table-uuid":        tableUUID,
			"location":          location,
			"last-updated-ms":   now,
			"properties":        req.Properties,
			"schemas":           []any{req.Schema},
			"current-schema-id": 0,
			"partition-specs":   []any{req.PartitionSpec},
			"default-spec-id":   0,
			"sort-orders":       []any{req.SortOrder},
			"default-sort-order-id": 0,
			"snapshots":         []any{},
			"snapshot-log":      []any{},
			"refs":              map[string]any{},
		}

		// Generate metadata location
		metadataLocation := fmt.Sprintf("%s/metadata/00000-%.10d-%s.metadata.json",
			location, now, tableUUID[:8])

		// Use pooled buffer for JSON serialization
		buf := pool.GetBuffer()
		defer pool.PutBuffer(buf)

		encoder := json.NewEncoder(buf)
		if err := encoder.Encode(metadata); err != nil {
			return internalError(c, "failed to marshal metadata", err)
		}
		metadataJSON := buf.Bytes()

		// Insert table
		_, err = database.Pool.Exec(c.Context(), `
			INSERT INTO tables (namespace_id, name, metadata_location, metadata)
			VALUES ($1, $2, $3, $4)
		`, namespaceID, req.Name, metadataLocation, metadataJSON)

		if err != nil {
			if IsDuplicateError(err) {
				return alreadyExistsError(c, "Table", fmt.Sprintf("%s.%s", strings.Join(namespaceName, "."), req.Name))
			}
			return internalError(c, "failed to create table", err)
		}

		// Record metrics
		metrics.RecordTableCreated(strings.Join(namespaceName, "."))

		// Log audit event
		event := events.NewEvent(events.TableCreated).
			WithNamespace(strings.Join(namespaceName, ".")).
			WithTable(req.Name)
		LogAudit(c, auditLogger, event, fiber.StatusOK)

		return c.JSON(LoadTableResponse{
			MetadataLocation: metadataLocation,
			Metadata:         metadata,
		})
	}
}

// LoadTable retrieves table metadata.
// GET /v1/{prefix}/namespaces/{namespace}/tables/{table}
func LoadTable(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))
		tableName := c.Params("table")

		// Single query with JOIN - eliminates N+1
		var metadataLocation string
		var metadata map[string]any
		err := database.Pool.QueryRow(c.Context(), `
			SELECT t.metadata_location, t.metadata FROM tables t
			JOIN namespaces n ON t.namespace_id = n.id
			WHERE n.name = $1 AND t.name = $2
		`, namespaceName, tableName).Scan(&metadataLocation, &metadata)

		if err == pgx.ErrNoRows {
			// Check if namespace exists to return appropriate error
			var nsExists bool
			_ = database.Pool.QueryRow(c.Context(), `
				SELECT EXISTS(SELECT 1 FROM namespaces WHERE name = $1)
			`, namespaceName).Scan(&nsExists)
			if !nsExists {
				return namespaceNotFound(c, namespaceName)
			}
			return tableNotFound(c, namespaceName, tableName)
		}
		if err != nil {
			return internalError(c, "failed to get table", err)
		}

		return c.JSON(LoadTableResponse{
			MetadataLocation: metadataLocation,
			Metadata:         metadata,
		})
	}
}

// CommitTable commits updates to a table.
// POST /v1/{prefix}/namespaces/{namespace}/tables/{table}
func CommitTable(database *db.DB, cfg *config.Config) fiber.Handler {
	// Build lock config from catalog settings
	lockCfg := db.LockConfig{
		Timeout:       cfg.Catalog.LockTimeout,
		RetryInterval: cfg.Catalog.LockRetryInterval,
		MaxRetries:    cfg.Catalog.MaxLockRetries,
	}

	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))
		tableName := c.Params("table")

		var req CommitTableRequest
		if err := c.BodyParser(&req); err != nil {
			return badRequest(c, "invalid request body")
		}

		// Result variables to capture from transaction
		var response LoadTableResponse
		var commitErr error

		// Execute within locked transaction with retry logic
		err := database.WithLock(c.Context(), lockCfg, func(tx pgx.Tx) error {
			// Get table with row lock
			var tableID string
			var metadataLocation string
			var metadata map[string]any
			err := tx.QueryRow(c.Context(), `
				SELECT t.id, t.metadata_location, t.metadata FROM tables t
				JOIN namespaces n ON t.namespace_id = n.id
				WHERE n.name = $1 AND t.name = $2
				FOR UPDATE OF t
			`, namespaceName, tableName).Scan(&tableID, &metadataLocation, &metadata)

			if errors.Is(err, pgx.ErrNoRows) {
				// Check if namespace exists to return appropriate error
				var nsExists bool
				//nolint:errcheck // Best effort check, default to table not found
				_ = tx.QueryRow(c.Context(), `
					SELECT EXISTS(SELECT 1 FROM namespaces WHERE name = $1)
				`, namespaceName).Scan(&nsExists)
				if !nsExists {
					commitErr = fmt.Errorf("namespace_not_found")
					return nil // Don't retry on not found
				}
				commitErr = fmt.Errorf("table_not_found")
				return nil // Don't retry on not found
			}
			if err != nil {
				return fmt.Errorf("failed to get table: %w", err)
			}

			// Validate requirements
			for _, r := range req.Requirements {
				if !validateRequirement(metadata, r) {
					commitErr = fmt.Errorf("requirement_failed:%s", r.Type)
					return nil // Don't retry on requirement failure
				}
			}

			// Apply updates
			now := time.Now().UnixMilli()
			metadata["last-updated-ms"] = now

			// Generate new metadata location
			tableUUID, ok := getStringFromMap(metadata, "table-uuid")
			if !ok {
				return fmt.Errorf("invalid metadata: missing table-uuid")
			}
			location, ok := getStringFromMap(metadata, "location")
			if !ok {
				return fmt.Errorf("invalid metadata: missing location")
			}
			newMetadataLocation := fmt.Sprintf("%s/metadata/%05d-%.10d-%s.metadata.json",
				location,
				getMetadataVersion(metadataLocation)+1,
				now,
				tableUUID[:8],
			)

			// Use pooled buffer for JSON serialization
			buf := pool.GetBuffer()
			defer pool.PutBuffer(buf)

			encoder := json.NewEncoder(buf)
			if err := encoder.Encode(metadata); err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}
			metadataJSON := buf.Bytes()

			// Update table
			_, err = tx.Exec(c.Context(), `
				UPDATE tables
				SET metadata_location = $1, metadata = $2, previous_metadata_location = $3
				WHERE id = $4
			`, newMetadataLocation, metadataJSON, metadataLocation, tableID)
			if err != nil {
				return fmt.Errorf("failed to update table: %w", err)
			}

			// Log the commit (best effort, don't fail on logging error)
			//nolint:errcheck
			_, _ = tx.Exec(c.Context(), `
				INSERT INTO commit_log (table_id, metadata_location)
				VALUES ($1, $2)
			`, tableID, newMetadataLocation)

			// Set response
			response = LoadTableResponse{
				MetadataLocation: newMetadataLocation,
				Metadata:         metadata,
			}

			return nil
		})

		// Handle errors
		if err != nil {
			if errors.Is(err, db.ErrLockTimeout) {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error": fiber.Map{
						"message": "Failed to acquire lock: concurrent modification in progress",
						"type":    "CommitFailedException",
						"code":    409,
					},
				})
			}
			return internalError(c, "failed to commit table", err)
		}

		// Handle business logic errors (not retryable)
		if commitErr != nil {
			errStr := commitErr.Error()
			if errStr == "namespace_not_found" {
				return namespaceNotFound(c, namespaceName)
			}
			if errStr == "table_not_found" {
				return tableNotFound(c, namespaceName, tableName)
			}
			if strings.HasPrefix(errStr, "requirement_failed:") {
				reqType := strings.TrimPrefix(errStr, "requirement_failed:")
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error": fiber.Map{
						"message": fmt.Sprintf("Requirement failed: %s", reqType),
						"type":    "CommitFailedException",
						"code":    409,
					},
				})
			}
		}

		// Record metrics
		metrics.RecordTableCommit(strings.Join(namespaceName, "."), tableName)

		return c.JSON(response)
	}
}

// DropTable deletes a table.
// DELETE /v1/{prefix}/namespaces/{namespace}/tables/{table}
func DropTable(database *db.DB, auditLogger *events.AuditLogger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))
		tableName := c.Params("table")
		purge := c.QueryBool("purgeRequested", false)

		// Single DELETE with JOIN - eliminates N+1
		result, err := database.Pool.Exec(c.Context(), `
			DELETE FROM tables t
			USING namespaces n
			WHERE t.namespace_id = n.id AND n.name = $1 AND t.name = $2
		`, namespaceName, tableName)

		if err != nil {
			return internalError(c, "failed to delete table", err)
		}

		if result.RowsAffected() == 0 {
			// Check if namespace exists to return appropriate error
			var nsExists bool
			_ = database.Pool.QueryRow(c.Context(), `
				SELECT EXISTS(SELECT 1 FROM namespaces WHERE name = $1)
			`, namespaceName).Scan(&nsExists)
			if !nsExists {
				return namespaceNotFound(c, namespaceName)
			}
			return tableNotFound(c, namespaceName, tableName)
		}

		// TODO: If purge is true, delete data files from storage
		_ = purge

		// Record metrics
		metrics.RecordTableDropped(strings.Join(namespaceName, "."))

		// Log audit event
		event := events.NewEvent(events.TableDropped).
			WithNamespace(strings.Join(namespaceName, ".")).
			WithTable(tableName)
		LogAudit(c, auditLogger, event, fiber.StatusNoContent)

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// TableExists checks if a table exists.
// HEAD /v1/{prefix}/namespaces/{namespace}/tables/{table}
func TableExists(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))
		tableName := c.Params("table")

		var exists bool
		err := database.Pool.QueryRow(c.Context(), `
			SELECT EXISTS(
				SELECT 1 FROM tables t
				JOIN namespaces n ON t.namespace_id = n.id
				WHERE n.name = $1 AND t.name = $2
			)
		`, namespaceName, tableName).Scan(&exists)

		if err != nil {
			return internalError(c, "failed to check table", err)
		}

		if !exists {
			return c.SendStatus(fiber.StatusNotFound)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// RegisterTable registers an existing table.
// POST /v1/{prefix}/namespaces/{namespace}/register
func RegisterTable(database *db.DB, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))

		var req RegisterTableRequest
		if err := c.BodyParser(&req); err != nil {
			return badRequest(c, "invalid request body")
		}

		if req.Name == "" {
			return badRequest(c, "table name is required")
		}

		if req.MetadataLocation == "" {
			return badRequest(c, "metadata-location is required")
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

		// TODO: Read metadata from the file system
		// For now, create a placeholder metadata
		tableUUID := uuid.New().String()
		metadata := map[string]any{
			"format-version":  2,
			"table-uuid":      tableUUID,
			"location":        strings.TrimSuffix(req.MetadataLocation, "/metadata"),
			"last-updated-ms": time.Now().UnixMilli(),
		}

		// Use pooled buffer for JSON serialization
		buf := pool.GetBuffer()
		defer pool.PutBuffer(buf)

		encoder := json.NewEncoder(buf)
		if err := encoder.Encode(metadata); err != nil {
			return internalError(c, "failed to marshal metadata", err)
		}
		metadataJSON := buf.Bytes()

		// Insert table
		_, err = database.Pool.Exec(c.Context(), `
			INSERT INTO tables (namespace_id, name, metadata_location, metadata)
			VALUES ($1, $2, $3, $4)
		`, namespaceID, req.Name, req.MetadataLocation, metadataJSON)

		if err != nil {
			if IsDuplicateError(err) {
				return alreadyExistsError(c, "Table", fmt.Sprintf("%s.%s", strings.Join(namespaceName, "."), req.Name))
			}
			return internalError(c, "failed to register table", err)
		}

		return c.JSON(LoadTableResponse{
			MetadataLocation: req.MetadataLocation,
			Metadata:         metadata,
		})
	}
}

// RenameTable renames a table.
// POST /v1/{prefix}/tables/rename
func RenameTable(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req RenameTableRequest
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

		// Rename table
		result, err := database.Pool.Exec(c.Context(), `
			UPDATE tables
			SET namespace_id = $1, name = $2
			WHERE namespace_id = $3 AND name = $4
		`, destNamespaceID, req.Destination.Name, sourceNamespaceID, req.Source.Name)

		if err != nil {
			if IsDuplicateError(err) {
				return alreadyExistsError(c, "Table", fmt.Sprintf("%s.%s",
					strings.Join(req.Destination.Namespace, "."), req.Destination.Name))
			}
			return internalError(c, "failed to rename table", err)
		}

		if result.RowsAffected() == 0 {
			return tableNotFound(c, req.Source.Namespace, req.Source.Name)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// tableNotFound returns a table not found error.
func tableNotFound(c *fiber.Ctx, namespace []string, table string) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": fiber.Map{
			"message": fmt.Sprintf("Table does not exist: %s.%s", strings.Join(namespace, "."), table),
			"type":    "NoSuchTableException",
			"code":    404,
		},
	})
}

// validateRequirement validates a commit requirement.
func validateRequirement(metadata map[string]any, req Requirement) bool {
	switch req.Type {
	case "assert-create":
		// Table must not exist (handled by caller)
		return true
	case "assert-table-uuid":
		return metadata["table-uuid"] == req.UUID
	case "assert-ref-snapshot-id":
		// Check ref snapshot ID
		refs, ok := metadata["refs"].(map[string]any)
		if !ok {
			return req.SnapshotID == nil
		}
		ref, ok := refs[req.Ref].(map[string]any)
		if !ok {
			return req.SnapshotID == nil
		}
		snapshotID, ok := ref["snapshot-id"].(float64)
		if !ok {
			return req.SnapshotID == nil
		}
		return req.SnapshotID != nil && int64(snapshotID) == *req.SnapshotID
	case "assert-last-assigned-field-id":
		lastFieldID, ok := metadata["last-column-id"].(float64)
		if !ok {
			return req.LastAssignedFieldID == nil
		}
		return req.LastAssignedFieldID != nil && int(lastFieldID) == *req.LastAssignedFieldID
	case "assert-current-schema-id":
		currentSchemaID, ok := metadata["current-schema-id"].(float64)
		if !ok {
			return req.CurrentSchemaID == nil
		}
		return req.CurrentSchemaID != nil && int(currentSchemaID) == *req.CurrentSchemaID
	case "assert-last-assigned-partition-id":
		// Check partition spec
		return true // Simplified
	case "assert-default-spec-id":
		defaultSpecID, ok := metadata["default-spec-id"].(float64)
		if !ok {
			return req.DefaultSpecID == nil
		}
		return req.DefaultSpecID != nil && int(defaultSpecID) == *req.DefaultSpecID
	case "assert-default-sort-order-id":
		defaultSortOrderID, ok := metadata["default-sort-order-id"].(float64)
		if !ok {
			return req.DefaultSortOrderID == nil
		}
		return req.DefaultSortOrderID != nil && int(defaultSortOrderID) == *req.DefaultSortOrderID
	default:
		return true
	}
}

// getMetadataVersion extracts version from metadata location.
func getMetadataVersion(location string) int {
	// Simple extraction - in production, parse properly
	return 0
}
