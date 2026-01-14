package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"

	"github.com/teamPaprika/bingsan/internal/config"
	"github.com/teamPaprika/bingsan/internal/db"
	"github.com/teamPaprika/bingsan/internal/metrics"
)

// CommitTransactionRequest is the request for committing a multi-table transaction.
type CommitTransactionRequest struct {
	TableChanges []TableChange `json:"table-changes"`
}

// TableChange represents changes to a single table in a transaction.
type TableChange struct {
	Identifier   TableIdentifier `json:"identifier"`
	Requirements []Requirement   `json:"requirements"`
	Updates      []Update        `json:"updates"`
}

// CommitTransactionResponse is the response for a multi-table transaction commit.
type CommitTransactionResponse struct {
	CommitResults []CommitResult `json:"commit-results"`
}

// CommitResult is the result of committing changes to a single table.
type CommitResult struct {
	Identifier       TableIdentifier `json:"identifier"`
	MetadataLocation string          `json:"metadata-location"`
	Metadata         map[string]any  `json:"metadata"`
}

// CommitTransaction commits atomic multi-table updates.
// POST /v1/{prefix}/transactions/commit
func CommitTransaction(database *db.DB, cfg *config.Config) fiber.Handler {
	// Build lock config from catalog settings
	lockCfg := db.LockConfig{
		Timeout:       cfg.Catalog.LockTimeout,
		RetryInterval: cfg.Catalog.LockRetryInterval,
		MaxRetries:    cfg.Catalog.MaxLockRetries,
	}

	return func(c *fiber.Ctx) error {
		var req CommitTransactionRequest
		if err := c.BodyParser(&req); err != nil {
			return badRequest(c, "invalid request body")
		}

		if len(req.TableChanges) == 0 {
			return badRequest(c, "at least one table change is required")
		}

		// Result variables to capture from transaction
		var results []CommitResult
		var txErr error

		// Execute within locked transaction with retry logic
		err := database.WithLock(c.Context(), lockCfg, func(tx pgx.Tx) error {
			results = make([]CommitResult, len(req.TableChanges))

			for i, change := range req.TableChanges {
				result, err := commitTableInTx(c.Context(), tx, cfg, change)
				if err != nil {
					// Check if it's a business logic error (not retryable)
					var fiberErr *fiber.Error
					if errors.As(err, &fiberErr) {
						txErr = err
						return nil // Don't retry on business errors
					}
					return err // Retry on lock/system errors
				}
				results[i] = *result
			}

			return nil
		})

		// Handle errors
		if err != nil {
			metrics.RecordTransactionCommit("failed")
			if errors.Is(err, db.ErrLockTimeout) {
				return fiber.NewError(fiber.StatusConflict,
					"Failed to acquire lock: concurrent modification in progress")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "failed to commit transaction: "+err.Error())
		}

		// Handle business logic errors (not retryable)
		if txErr != nil {
			metrics.RecordTransactionCommit("failed")
			return txErr
		}

		// Record metrics
		metrics.RecordTransactionCommit("success")

		return c.JSON(CommitTransactionResponse{
			CommitResults: results,
		})
	}
}

// commitTableInTx commits changes to a single table within a transaction.
func commitTableInTx(ctx context.Context, tx pgx.Tx, cfg *config.Config, change TableChange) (*CommitResult, error) {
	// Get namespace ID
	var namespaceID string
	err := tx.QueryRow(ctx, `
		SELECT id FROM namespaces WHERE name = $1
	`, change.Identifier.Namespace).Scan(&namespaceID)

	if err == pgx.ErrNoRows {
		return nil, fiber.NewError(fiber.StatusNotFound,
			fmt.Sprintf("Namespace does not exist: %s", strings.Join(change.Identifier.Namespace, ".")))
	}
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "failed to get namespace: "+err.Error())
	}

	// Get current table state with lock
	var tableID string
	var metadataLocation string
	var metadata map[string]any
	err = tx.QueryRow(ctx, `
		SELECT id, metadata_location, metadata FROM tables
		WHERE namespace_id = $1 AND name = $2
		FOR UPDATE
	`, namespaceID, change.Identifier.Name).Scan(&tableID, &metadataLocation, &metadata)

	if err == pgx.ErrNoRows {
		return nil, fiber.NewError(fiber.StatusNotFound,
			fmt.Sprintf("Table does not exist: %s.%s",
				strings.Join(change.Identifier.Namespace, "."), change.Identifier.Name))
	}
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "failed to get table: "+err.Error())
	}

	// Validate requirements
	for _, req := range change.Requirements {
		if !validateRequirement(metadata, req) {
			return nil, fiber.NewError(fiber.StatusConflict,
				fmt.Sprintf("Requirement failed for table %s.%s: %s",
					strings.Join(change.Identifier.Namespace, "."), change.Identifier.Name, req.Type))
		}
	}

	// TODO: Apply updates to metadata
	// For now, just return current state

	return &CommitResult{
		Identifier:       change.Identifier,
		MetadataLocation: metadataLocation,
		Metadata:         metadata,
	}, nil
}
