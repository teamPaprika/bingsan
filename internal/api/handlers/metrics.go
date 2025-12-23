package handlers

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

// MetricsReport represents a metrics report from a client.
type MetricsReport struct {
	TableName       string         `json:"table-name"`
	SnapshotID      *int64         `json:"snapshot-id,omitempty"`
	Filter          map[string]any `json:"filter,omitempty"`
	SchemaID        *int           `json:"schema-id,omitempty"`
	ProjectedFieldIDs []int        `json:"projected-field-ids,omitempty"`
	ProjectedFieldNames []string   `json:"projected-field-names,omitempty"`
	Metrics         ScanMetrics    `json:"metrics,omitempty"`
}

// ScanMetrics contains scan-related metrics.
type ScanMetrics struct {
	TotalPlanningDuration  *int64          `json:"total-planning-duration,omitempty"`
	ResultDataFiles        *int64          `json:"result-data-files,omitempty"`
	ResultDeleteFiles      *int64          `json:"result-delete-files,omitempty"`
	TotalDataManifests     *int64          `json:"total-data-manifests,omitempty"`
	TotalDeleteManifests   *int64          `json:"total-delete-manifests,omitempty"`
	ScannedDataManifests   *int64          `json:"scanned-data-manifests,omitempty"`
	SkippedDataManifests   *int64          `json:"skipped-data-manifests,omitempty"`
	TotalFileSizeInBytes   *int64          `json:"total-file-size-in-bytes,omitempty"`
	TotalDeleteFileSizeInBytes *int64      `json:"total-delete-file-size-in-bytes,omitempty"`
	SkippedDataFiles       *int64          `json:"skipped-data-files,omitempty"`
	SkippedDeleteFiles     *int64          `json:"skipped-delete-files,omitempty"`
	IndexedDeleteFiles     *int64          `json:"indexed-delete-files,omitempty"`
	EqualityDeleteFiles    *int64          `json:"equality-delete-files,omitempty"`
	PositionalDeleteFiles  *int64          `json:"positional-delete-files,omitempty"`
	Extra                  map[string]string `json:"extra,omitempty"`
}

// CommitMetrics contains commit-related metrics.
type CommitMetrics struct {
	TotalDuration        *int64            `json:"total-duration,omitempty"`
	Attempts             *int64            `json:"attempts,omitempty"`
	AddedDataFiles       *int64            `json:"added-data-files,omitempty"`
	RemovedDataFiles     *int64            `json:"removed-data-files,omitempty"`
	TotalDataFiles       *int64            `json:"total-data-files,omitempty"`
	AddedDeleteFiles     *int64            `json:"added-delete-files,omitempty"`
	AddedEqDeleteFiles   *int64            `json:"added-equality-delete-files,omitempty"`
	AddedPosDeleteFiles  *int64            `json:"added-positional-delete-files,omitempty"`
	RemovedDeleteFiles   *int64            `json:"removed-delete-files,omitempty"`
	RemovedEqDeleteFiles *int64            `json:"removed-equality-delete-files,omitempty"`
	RemovedPosDeleteFiles *int64           `json:"removed-positional-delete-files,omitempty"`
	TotalDeleteFiles     *int64            `json:"total-delete-files,omitempty"`
	AddedRecords         *int64            `json:"added-records,omitempty"`
	DeletedRecords       *int64            `json:"deleted-records,omitempty"`
	TotalRecords         *int64            `json:"total-records,omitempty"`
	AddedFileSizeInBytes *int64            `json:"added-file-size-in-bytes,omitempty"`
	RemovedFileSizeInBytes *int64          `json:"removed-file-size-in-bytes,omitempty"`
	TotalFileSizeInBytes *int64            `json:"total-file-size-in-bytes,omitempty"`
	Extra                map[string]string `json:"extra,omitempty"`
}

// ReportMetrics handles metrics reporting from clients.
// POST /v1/{prefix}/namespaces/{namespace}/tables/{table}/metrics
//
// This is an optional endpoint that allows clients to report
// scan and commit metrics back to the catalog for monitoring.
func ReportMetrics() fiber.Handler {
	return func(c *fiber.Ctx) error {
		namespaceName := parseNamespace(c.Params("namespace"))
		tableName := c.Params("table")

		var report MetricsReport
		if err := c.BodyParser(&report); err != nil {
			// Don't fail on invalid metrics - just log and continue
			slog.Warn("failed to parse metrics report",
				"namespace", namespaceName,
				"table", tableName,
				"error", err,
			)
			return c.SendStatus(fiber.StatusNoContent)
		}

		// Log metrics for observability
		// In production, you might want to:
		// - Store metrics in a time-series database
		// - Send to a metrics service (Prometheus, DataDog, etc.)
		// - Aggregate for dashboards
		slog.Info("metrics reported",
			"namespace", namespaceName,
			"table", tableName,
			"snapshot_id", report.SnapshotID,
			"result_data_files", report.Metrics.ResultDataFiles,
			"result_delete_files", report.Metrics.ResultDeleteFiles,
			"total_file_size_bytes", report.Metrics.TotalFileSizeInBytes,
			"planning_duration_ms", report.Metrics.TotalPlanningDuration,
		)

		// Return 204 No Content as per spec
		return c.SendStatus(fiber.StatusNoContent)
	}
}
