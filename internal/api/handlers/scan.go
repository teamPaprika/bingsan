package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kimuyb/bingsan/internal/db"
)

// ScanPlanRequest is the request for submitting a scan plan.
type ScanPlanRequest struct {
	SnapshotID   *int64            `json:"snapshot-id,omitempty"`
	Select       []string          `json:"select,omitempty"`
	Filter       map[string]any    `json:"filter,omitempty"`
	CaseSensitive bool             `json:"case-sensitive,omitempty"`
	UseSnapshot  bool              `json:"use-snapshot-schema,omitempty"`
	StartSnapshotID *int64         `json:"start-snapshot-id,omitempty"`
	EndSnapshotID *int64           `json:"end-snapshot-id,omitempty"`
}

// ScanPlanResponse is the response for a scan plan.
type ScanPlanResponse struct {
	PlanID string `json:"plan-id"`
	Status string `json:"status"`
}

// ScanPlanResult is the result of a completed scan plan.
type ScanPlanResult struct {
	PlanID     string         `json:"plan-id"`
	Status     string         `json:"status"`
	ScanTasks  []ScanTask     `json:"scan-tasks,omitempty"`
	NextToken  string         `json:"next-page-token,omitempty"`
}

// ScanTask represents a single scan task.
type ScanTask struct {
	TaskID       string         `json:"task-id"`
	DataFiles    []DataFile     `json:"data-files,omitempty"`
	DeleteFiles  []DeleteFile   `json:"delete-files,omitempty"`
	Residuals    map[string]any `json:"residuals,omitempty"`
}

// DataFile represents a data file in a scan task.
type DataFile struct {
	FilePath       string         `json:"file-path"`
	FileFormat     string         `json:"file-format"`
	RecordCount    int64          `json:"record-count"`
	FileSizeBytes  int64          `json:"file-size-in-bytes"`
	ColumnSizes    map[int]int64  `json:"column-sizes,omitempty"`
	ValueCounts    map[int]int64  `json:"value-counts,omitempty"`
	NullValueCounts map[int]int64 `json:"null-value-counts,omitempty"`
	NanValueCounts  map[int]int64 `json:"nan-value-counts,omitempty"`
	LowerBounds    map[int]any    `json:"lower-bounds,omitempty"`
	UpperBounds    map[int]any    `json:"upper-bounds,omitempty"`
}

// DeleteFile represents a delete file in a scan task.
type DeleteFile struct {
	FilePath       string `json:"file-path"`
	FileFormat     string `json:"file-format"`
	RecordCount    int64  `json:"record-count"`
	FileSizeBytes  int64  `json:"file-size-in-bytes"`
}

// FetchTasksRequest is the request for fetching plan tasks.
type FetchTasksRequest struct {
	PlanID    string `json:"plan-id"`
	PageToken string `json:"page-token,omitempty"`
}

// SubmitScanPlan submits a scan for server-side planning.
// POST /v1/{prefix}/namespaces/{namespace}/tables/{table}/plan
func SubmitScanPlan(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))
		tableName := c.Params("table")

		var req ScanPlanRequest
		if err := c.BodyParser(&req); err != nil {
			return badRequest(c, "invalid request body")
		}

		// Get table ID
		var tableID string
		err := database.Pool.QueryRow(c.Context(), `
			SELECT t.id FROM tables t
			JOIN namespaces n ON t.namespace_id = n.id
			WHERE n.name = $1 AND t.name = $2
		`, namespaceName, tableName).Scan(&tableID)

		if err == pgx.ErrNoRows {
			return tableNotFound(c, namespaceName, tableName)
		}
		if err != nil {
			return internalError(c, "failed to get table", err)
		}

		// Create scan plan
		planID := uuid.New().String()
		expiresAt := time.Now().Add(1 * time.Hour)

		planData := map[string]any{
			"snapshot-id":   req.SnapshotID,
			"select":        req.Select,
			"filter":        req.Filter,
			"case-sensitive": req.CaseSensitive,
		}

		_, err = database.Pool.Exec(c.Context(), `
			INSERT INTO scan_plans (id, table_id, plan_data, status, expires_at)
			VALUES ($1, $2, $3, 'pending', $4)
		`, planID, tableID, planData, expiresAt)

		if err != nil {
			return internalError(c, "failed to create scan plan", err)
		}

		// TODO: Start async scan planning task
		// For now, mark as completed immediately
		_, _ = database.Pool.Exec(c.Context(), `
			UPDATE scan_plans SET status = 'completed' WHERE id = $1
		`, planID)

		return c.Status(fiber.StatusAccepted).JSON(ScanPlanResponse{
			PlanID: planID,
			Status: "pending",
		})
	}
}

// GetScanPlan fetches planning results.
// GET /v1/{prefix}/namespaces/{namespace}/tables/{table}/plan/{plan-id}
func GetScanPlan(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))
		tableName := c.Params("table")
		planID := c.Params("planId")

		// Verify table exists and plan belongs to it
		var status string
		var planData map[string]any
		err := database.Pool.QueryRow(c.Context(), `
			SELECT sp.status, sp.plan_data FROM scan_plans sp
			JOIN tables t ON sp.table_id = t.id
			JOIN namespaces n ON t.namespace_id = n.id
			WHERE n.name = $1 AND t.name = $2 AND sp.id = $3
		`, namespaceName, tableName, planID).Scan(&status, &planData)

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

		result := ScanPlanResult{
			PlanID: planID,
			Status: status,
		}

		// If completed, include scan tasks
		if status == "completed" {
			// TODO: Generate actual scan tasks from table metadata
			result.ScanTasks = []ScanTask{}
		}

		return c.JSON(result)
	}
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
func FetchPlanTasks(database *db.DB) fiber.Handler {
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

		// Get plan status
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

		// TODO: Return actual scan tasks
		return c.JSON(ScanPlanResult{
			PlanID:    req.PlanID,
			Status:    status,
			ScanTasks: []ScanTask{},
		})
	}
}
