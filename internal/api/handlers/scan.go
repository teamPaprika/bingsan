package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kimuyb/bingsan/internal/db"
	"github.com/kimuyb/bingsan/internal/scan"
)

// ScanPlanRequest is the request for submitting a scan plan.
type ScanPlanRequest struct {
	SnapshotID      *int64         `json:"snapshot-id,omitempty"`
	Select          []string       `json:"select,omitempty"`
	Filter          map[string]any `json:"filter,omitempty"`
	CaseSensitive   bool           `json:"case-sensitive,omitempty"`
	UseSnapshot     bool           `json:"use-snapshot-schema,omitempty"`
	StartSnapshotID *int64         `json:"start-snapshot-id,omitempty"`
	EndSnapshotID   *int64         `json:"end-snapshot-id,omitempty"`
}

// ScanPlanResponse is the response for a scan plan.
type ScanPlanResponse struct {
	PlanID         string                 `json:"plan-id"`
	Status         string                 `json:"status"`
	FileScanTasks  []scan.FileScanTask    `json:"file-scan-tasks,omitempty"`
	DeleteFiles    []scan.DeleteFileInfo  `json:"delete-files,omitempty"`
	PlanTasks      []string               `json:"plan-tasks,omitempty"`
}

// FetchTasksRequest is the request for fetching plan tasks.
type FetchTasksRequest struct {
	PlanID    string `json:"plan-id"`
	PlanTask  string `json:"plan-task,omitempty"`
	PageToken string `json:"page-token,omitempty"`
}

// PlannerGetter is a function that returns the scan planner.
type PlannerGetter func() *scan.Planner

// SubmitScanPlan submits a scan for server-side planning.
// POST /v1/{prefix}/namespaces/{namespace}/tables/{table}/plan
func SubmitScanPlan(database *db.DB, getPlanner PlannerGetter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))
		tableName := c.Params("table")

		var req ScanPlanRequest
		if err := c.BodyParser(&req); err != nil {
			return badRequest(c, "invalid request body")
		}

		// Check if planner is available
		planner := getPlanner()
		if planner == nil {
			// Fall back to stub behavior if planner not configured
			return submitScanPlanStub(c, database, namespaceName, tableName, req)
		}

		// Parse filter expression
		var filter *scan.Expression
		if req.Filter != nil {
			var err error
			filter, err = scan.ParseExpression(req.Filter)
			if err != nil {
				return badRequest(c, fmt.Sprintf("invalid filter expression: %v", err))
			}
		}

		// Execute planning
		result, err := planner.Plan(c.Context(), scan.PlanRequest{
			Namespace:     namespaceName,
			TableName:     tableName,
			SnapshotID:    req.SnapshotID,
			Filter:        filter,
			Select:        req.Select,
			CaseSensitive: req.CaseSensitive,
		})
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return tableNotFound(c, namespaceName, tableName)
			}
			return internalError(c, "scan planning failed", err)
		}

		// Return appropriate response based on sync vs async
		response := ScanPlanResponse{
			PlanID: result.PlanID,
			Status: result.Status,
		}

		if result.Status == "completed" {
			response.FileScanTasks = result.FileScanTasks
			response.DeleteFiles = result.DeleteFiles
			return c.Status(fiber.StatusOK).JSON(response)
		}

		// Async plan submitted
		return c.Status(fiber.StatusAccepted).JSON(response)
	}
}

// submitScanPlanStub is the fallback when planner is not configured.
func submitScanPlanStub(c *fiber.Ctx, database *db.DB, namespace []string, tableName string, req ScanPlanRequest) error {
	// Get table ID
	var tableID string
	err := database.Pool.QueryRow(c.Context(), `
		SELECT t.id FROM tables t
		JOIN namespaces n ON t.namespace_id = n.id
		WHERE n.name = $1 AND t.name = $2
	`, namespace, tableName).Scan(&tableID)

	if err == pgx.ErrNoRows {
		return tableNotFound(c, namespace, tableName)
	}
	if err != nil {
		return internalError(c, "failed to get table", err)
	}

	// Create stub scan plan
	planID := uuid.New().String()
	expiresAt := time.Now().Add(1 * time.Hour)

	planData := map[string]any{
		"snapshot-id":    req.SnapshotID,
		"select":         req.Select,
		"filter":         req.Filter,
		"case-sensitive": req.CaseSensitive,
	}

	_, err = database.Pool.Exec(c.Context(), `
		INSERT INTO scan_plans (id, table_id, plan_data, status, expires_at)
		VALUES ($1, $2, $3, 'completed', $4)
	`, planID, tableID, planData, expiresAt)

	if err != nil {
		return internalError(c, "failed to create scan plan", err)
	}

	return c.Status(fiber.StatusOK).JSON(ScanPlanResponse{
		PlanID:        planID,
		Status:        "completed",
		FileScanTasks: []scan.FileScanTask{},
	})
}

// GetScanPlan fetches planning results.
// GET /v1/{prefix}/namespaces/{namespace}/tables/{table}/plan/{plan-id}
func GetScanPlan(database *db.DB, getPlanner PlannerGetter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))
		tableName := c.Params("table")
		planID := c.Params("planId")

		// Check if planner is available
		planner := getPlanner()
		if planner == nil {
			return getScanPlanStub(c, database, namespaceName, tableName, planID)
		}

		// Verify the plan belongs to this table
		var tableID string
		err := database.Pool.QueryRow(c.Context(), `
			SELECT sp.table_id FROM scan_plans sp
			JOIN tables t ON sp.table_id = t.id
			JOIN namespaces n ON t.namespace_id = n.id
			WHERE n.name = $1 AND t.name = $2 AND sp.id = $3
		`, namespaceName, tableName, planID).Scan(&tableID)

		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": fiber.Map{
					"message": fmt.Sprintf("Scan plan not found: %s", planID),
					"type":    "NoSuchPlanException",
					"code":    404,
				},
			})
		}
		if err != nil {
			return internalError(c, "failed to verify scan plan", err)
		}

		// Get plan result
		result, err := planner.GetPlanResult(c.Context(), planID)
		if err != nil {
			return internalError(c, "failed to get plan result", err)
		}

		response := ScanPlanResponse{
			PlanID:        result.PlanID,
			Status:        result.Status,
			FileScanTasks: result.FileScanTasks,
			DeleteFiles:   result.DeleteFiles,
			PlanTasks:     result.PlanTasks,
		}

		return c.JSON(response)
	}
}

// getScanPlanStub is the fallback when planner is not configured.
func getScanPlanStub(c *fiber.Ctx, database *db.DB, namespace []string, tableName, planID string) error {
	var status string
	err := database.Pool.QueryRow(c.Context(), `
		SELECT sp.status FROM scan_plans sp
		JOIN tables t ON sp.table_id = t.id
		JOIN namespaces n ON t.namespace_id = n.id
		WHERE n.name = $1 AND t.name = $2 AND sp.id = $3
	`, namespace, tableName, planID).Scan(&status)

	if err == pgx.ErrNoRows {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{
				"message": fmt.Sprintf("Scan plan not found: %s", planID),
				"type":    "NoSuchPlanException",
				"code":    404,
			},
		})
	}
	if err != nil {
		return internalError(c, "failed to get scan plan", err)
	}

	return c.JSON(ScanPlanResponse{
		PlanID:        planID,
		Status:        status,
		FileScanTasks: []scan.FileScanTask{},
	})
}

// CancelScanPlan cancels a scan planning request.
// DELETE /v1/{prefix}/namespaces/{namespace}/tables/{table}/plan/{plan-id}
func CancelScanPlan(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))
		tableName := c.Params("table")
		planID := c.Params("planId")

		// Update plan status to cancelled
		result, err := database.Pool.Exec(c.Context(), `
			UPDATE scan_plans sp
			SET status = 'cancelled'
			FROM tables t, namespaces n
			WHERE sp.table_id = t.id
			AND t.namespace_id = n.id
			AND n.name = $1 AND t.name = $2 AND sp.id = $3
			AND sp.status IN ('pending', 'running')
		`, namespaceName, tableName, planID)

		if err != nil {
			return internalError(c, "failed to cancel scan plan", err)
		}

		if result.RowsAffected() == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": fiber.Map{
					"message": fmt.Sprintf("Scan plan not found or already completed: %s", planID),
					"type":    "NoSuchPlanException",
					"code":    404,
				},
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// FetchPlanTasks fetches result tasks for a plan.
// POST /v1/{prefix}/namespaces/{namespace}/tables/{table}/tasks
func FetchPlanTasks(database *db.DB, getPlanner PlannerGetter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))
		tableName := c.Params("table")

		var req FetchTasksRequest
		if err := c.BodyParser(&req); err != nil {
			return badRequest(c, "invalid request body")
		}

		if req.PlanID == "" {
			return badRequest(c, "plan-id is required")
		}

		// Verify the plan belongs to this table and get its status
		var status string
		err := database.Pool.QueryRow(c.Context(), `
			SELECT sp.status FROM scan_plans sp
			JOIN tables t ON sp.table_id = t.id
			JOIN namespaces n ON t.namespace_id = n.id
			WHERE n.name = $1 AND t.name = $2 AND sp.id = $3
		`, namespaceName, tableName, req.PlanID).Scan(&status)

		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": fiber.Map{
					"message": fmt.Sprintf("Scan plan not found: %s", req.PlanID),
					"type":    "NoSuchPlanException",
					"code":    404,
				},
			})
		}
		if err != nil {
			return internalError(c, "failed to get scan plan", err)
		}

		if status != "completed" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{
					"message": fmt.Sprintf("Scan plan not ready: %s (status: %s)", req.PlanID, status),
					"type":    "PlanNotReadyException",
					"code":    400,
				},
			})
		}

		// If no plan task specified, return full result
		planner := getPlanner()
		if req.PlanTask == "" || planner == nil {
			if planner == nil {
				return c.JSON(ScanPlanResponse{
					PlanID:        req.PlanID,
					Status:        status,
					FileScanTasks: []scan.FileScanTask{},
				})
			}

			result, err := planner.GetPlanResult(c.Context(), req.PlanID)
			if err != nil {
				return internalError(c, "failed to get plan result", err)
			}

			return c.JSON(ScanPlanResponse{
				PlanID:        result.PlanID,
				Status:        result.Status,
				FileScanTasks: result.FileScanTasks,
				DeleteFiles:   result.DeleteFiles,
			})
		}

		// Resolve plan task to file scan tasks
		tasks, deletes, err := planner.FetchTasksForPlanTask(c.Context(), req.PlanID, req.PlanTask)
		if err != nil {
			return internalError(c, "failed to fetch tasks", err)
		}

		return c.JSON(ScanPlanResponse{
			PlanID:        req.PlanID,
			Status:        "completed",
			FileScanTasks: tasks,
			DeleteFiles:   deletes,
		})
	}
}

// Note: parseNamespace is defined in namespace.go
