package scan

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kimuyb/bingsan/internal/avro"
	"github.com/kimuyb/bingsan/internal/db"
	"github.com/kimuyb/bingsan/internal/metrics"
	"github.com/kimuyb/bingsan/internal/storage"
)

// Config holds scan planner configuration.
type Config struct {
	// AsyncThreshold is the number of files above which async planning is used.
	AsyncThreshold int

	// MaxConcurrentManifests limits parallel manifest reads.
	MaxConcurrentManifests int

	// PlanTaskBatchSize is files per plan-task for async plans.
	PlanTaskBatchSize int

	// PlanExpiry is how long scan plans are kept before cleanup.
	PlanExpiry time.Duration
}

// DefaultConfig returns default planner configuration.
func DefaultConfig() Config {
	return Config{
		AsyncThreshold:         100,
		MaxConcurrentManifests: 10,
		PlanTaskBatchSize:      50,
		PlanExpiry:             1 * time.Hour,
	}
}

// Planner orchestrates scan planning operations.
type Planner struct {
	storage storage.Client
	db      *db.DB
	config  Config
}

// NewPlanner creates a new scan planner.
func NewPlanner(storageClient storage.Client, database *db.DB, cfg Config) *Planner {
	return &Planner{
		storage: storageClient,
		db:      database,
		config:  cfg,
	}
}

// Plan executes scan planning for a table.
// For small tables, returns results immediately.
// For large tables, creates an async plan and returns the plan ID.
func (p *Planner) Plan(ctx context.Context, req PlanRequest) (*PlanResult, error) {
	// Get table metadata
	metadata, err := p.getTableMetadata(ctx, req.Namespace, req.TableName)
	if err != nil {
		return nil, err
	}

	// Determine snapshot to scan
	snapshotID, snapshot, err := p.resolveSnapshot(metadata, req.SnapshotID)
	if err != nil {
		return nil, err
	}

	// Get manifest list path from snapshot
	manifestListPath, ok := snapshot["manifest-list"].(string)
	if !ok || manifestListPath == "" {
		// Empty table (no snapshot or no manifest list)
		return &PlanResult{
			PlanID:        uuid.New().String(),
			Status:        "completed",
			FileScanTasks: []FileScanTask{},
		}, nil
	}

	// Read manifest list to estimate file count
	manifestList, err := p.readManifestList(ctx, manifestListPath)
	if err != nil {
		return nil, fmt.Errorf("reading manifest list: %w", err)
	}

	estimatedFiles := manifestList.TotalDataFiles()

	slog.Debug("scan planning",
		"table", req.TableName,
		"snapshot", snapshotID,
		"manifests", len(manifestList.Entries),
		"estimated_files", estimatedFiles,
	)

	// Decide sync vs async based on file count
	if int(estimatedFiles) <= p.config.AsyncThreshold {
		return p.planSync(ctx, req, metadata, manifestList)
	}

	return p.planAsync(ctx, req, snapshotID, manifestListPath, estimatedFiles)
}

// planSync executes planning synchronously for small tables.
func (p *Planner) planSync(ctx context.Context, req PlanRequest, metadata map[string]any, manifestList *avro.ManifestList) (*PlanResult, error) {
	start := time.Now()
	defer func() {
		metrics.RecordScanPlanDuration("sync", time.Since(start).Seconds())
		metrics.RecordScanPlanMode("sync")
	}()

	// Get schema for filter evaluation
	schemas, _ := metadata["schemas"].([]any)
	currentSchemaID, _ := metadata["current-schema-id"].(float64)
	schema := p.getSchema(schemas, int(currentSchemaID))

	// Create evaluator for filter pruning
	evaluator := NewEvaluator(schema, req.CaseSensitive)

	// Read all manifests in parallel
	dataManifests := manifestList.DataManifests()
	deleteManifests := manifestList.DeleteManifests()

	var dataFiles []avro.DataFile
	var deleteFiles []avro.DataFile
	var filesPruned int

	// Read data manifests
	dataResults := p.readManifestsParallel(ctx, dataManifests)
	metrics.RecordScanManifestsRead(len(dataManifests))

	for _, manifest := range dataResults {
		for _, df := range manifest.LiveDataFiles() {
			if req.Filter != nil && !evaluator.ShouldIncludeFile(req.Filter, &df) {
				filesPruned++
				continue
			}
			dataFiles = append(dataFiles, df)
		}
	}

	// Read delete manifests
	deleteResults := p.readManifestsParallel(ctx, deleteManifests)
	metrics.RecordScanManifestsRead(len(deleteManifests))

	for _, manifest := range deleteResults {
		deleteFiles = append(deleteFiles, manifest.LiveDeleteFiles()...)
	}

	// Record file metrics
	metrics.RecordScanFiles(len(dataFiles), filesPruned)
	metrics.RecordScanPlan("completed")

	// Build result
	result := &PlanResult{
		PlanID:      uuid.New().String(),
		Status:      "completed",
		FileCount:   len(dataFiles),
		FilesPruned: filesPruned,
	}

	// Convert to API types
	for _, df := range dataFiles {
		specID := 0 // TODO: get from manifest entry
		task := FileScanTask{
			DataFile: NewDataFileInfo(&df, specID),
		}
		result.FileScanTasks = append(result.FileScanTasks, task)
	}

	for _, df := range deleteFiles {
		specID := 0
		result.DeleteFiles = append(result.DeleteFiles, NewDeleteFileInfo(&df, specID))
	}

	slog.Info("scan plan completed (sync)",
		"plan_id", result.PlanID,
		"files", result.FileCount,
		"pruned", result.FilesPruned,
	)

	return result, nil
}

// planAsync creates an async plan for large tables.
func (p *Planner) planAsync(ctx context.Context, req PlanRequest, snapshotID int64, manifestListPath string, estimatedFiles int64) (*PlanResult, error) {
	planID := uuid.New().String()
	expiresAt := time.Now().Add(p.config.PlanExpiry)

	// Store plan request data
	planData := map[string]any{
		"namespace":       req.Namespace,
		"table":           req.TableName,
		"snapshot_id":     snapshotID,
		"manifest_list":   manifestListPath,
		"filter":          req.Filter,
		"select":          req.Select,
		"case_sensitive":  req.CaseSensitive,
		"estimated_files": estimatedFiles,
	}

	_, err := p.db.Pool.Exec(ctx, `
		INSERT INTO scan_plans (id, table_id, plan_data, status, expires_at)
		SELECT $1, t.id, $2, 'pending', $3
		FROM tables t
		JOIN namespaces n ON t.namespace_id = n.id
		WHERE n.name = $4 AND t.name = $5
	`, planID, planData, expiresAt, req.Namespace, req.TableName)

	if err != nil {
		return nil, fmt.Errorf("creating scan plan: %w", err)
	}

	// Record metrics
	metrics.RecordScanPlanMode("async")
	metrics.RecordScanPlan("submitted")

	slog.Info("scan plan created (async)",
		"plan_id", planID,
		"estimated_files", estimatedFiles,
	)

	return &PlanResult{
		PlanID: planID,
		Status: "submitted",
	}, nil
}

// CompletePlan finishes an async plan (called by background worker).
func (p *Planner) CompletePlan(ctx context.Context, planID string) error {
	start := time.Now()
	metrics.IncScanActivePlans()
	defer func() {
		metrics.DecScanActivePlans()
		metrics.RecordScanPlanDuration("async", time.Since(start).Seconds())
	}()

	// Get plan data
	var planData map[string]any
	var tableID string
	err := p.db.Pool.QueryRow(ctx, `
		UPDATE scan_plans
		SET status = 'running', started_at = NOW()
		WHERE id = $1 AND status = 'pending'
		RETURNING table_id, plan_data
	`, planID).Scan(&tableID, &planData)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil // Plan already processed or doesn't exist
		}
		return fmt.Errorf("getting plan: %w", err)
	}

	// Extract plan parameters
	namespace, _ := planData["namespace"].([]any)
	namespaceStr := make([]string, len(namespace))
	for i, n := range namespace {
		namespaceStr[i], _ = n.(string)
	}
	tableName, _ := planData["table"].(string)
	manifestListPath, _ := planData["manifest_list"].(string)
	caseSensitive, _ := planData["case_sensitive"].(bool)

	var filter *Expression
	if filterData, ok := planData["filter"].(map[string]any); ok {
		filter, _ = ParseExpression(filterData)
	}

	// Get table metadata for schema
	metadata, err := p.getTableMetadata(ctx, namespaceStr, tableName)
	if err != nil {
		p.failPlan(ctx, planID, err.Error())
		return err
	}

	// Read manifest list
	manifestList, err := p.readManifestList(ctx, manifestListPath)
	if err != nil {
		p.failPlan(ctx, planID, err.Error())
		return fmt.Errorf("reading manifest list: %w", err)
	}

	// Get schema for filter evaluation
	schemas, _ := metadata["schemas"].([]any)
	currentSchemaID, _ := metadata["current-schema-id"].(float64)
	schema := p.getSchema(schemas, int(currentSchemaID))
	evaluator := NewEvaluator(schema, caseSensitive)

	// Process manifests
	dataManifests := manifestList.DataManifests()
	deleteManifests := manifestList.DeleteManifests()

	var dataFiles []avro.DataFile
	var deleteFiles []avro.DataFile
	var filesPruned int

	// Read data manifests
	dataResults := p.readManifestsParallel(ctx, dataManifests)
	metrics.RecordScanManifestsRead(len(dataManifests))

	for _, manifest := range dataResults {
		for _, df := range manifest.LiveDataFiles() {
			if filter != nil && !evaluator.ShouldIncludeFile(filter, &df) {
				filesPruned++
				continue
			}
			dataFiles = append(dataFiles, df)
		}
	}

	// Read delete manifests
	deleteResults := p.readManifestsParallel(ctx, deleteManifests)
	metrics.RecordScanManifestsRead(len(deleteManifests))

	for _, manifest := range deleteResults {
		deleteFiles = append(deleteFiles, manifest.LiveDeleteFiles()...)
	}

	// Record file metrics
	metrics.RecordScanFiles(len(dataFiles), filesPruned)

	// Build result data
	resultData := map[string]any{
		"file_scan_tasks": p.buildFileScanTasks(dataFiles),
		"delete_files":    p.buildDeleteFiles(deleteFiles),
		"plan_tasks":      p.buildPlanTasks(dataFiles),
	}

	// Store result
	_, err = p.db.Pool.Exec(ctx, `
		UPDATE scan_plans
		SET status = 'completed',
		    completed_at = NOW(),
		    file_count = $2,
		    files_pruned = $3,
		    result_data = $4
		WHERE id = $1
	`, planID, len(dataFiles), filesPruned, resultData)

	if err != nil {
		return fmt.Errorf("storing plan result: %w", err)
	}

	// Record completion metric
	metrics.RecordScanPlan("completed")

	slog.Info("scan plan completed (async)",
		"plan_id", planID,
		"files", len(dataFiles),
		"pruned", filesPruned,
	)

	return nil
}

// GetPlanResult retrieves the result of a completed plan.
func (p *Planner) GetPlanResult(ctx context.Context, planID string) (*PlanResult, error) {
	var status string
	var resultData map[string]any
	var fileCount, filesPruned int

	err := p.db.Pool.QueryRow(ctx, `
		SELECT status, COALESCE(result_data, '{}'::jsonb), COALESCE(file_count, 0), COALESCE(files_pruned, 0)
		FROM scan_plans
		WHERE id = $1
	`, planID).Scan(&status, &resultData, &fileCount, &filesPruned)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("plan not found: %s", planID)
		}
		return nil, fmt.Errorf("getting plan: %w", err)
	}

	result := &PlanResult{
		PlanID:      planID,
		Status:      status,
		FileCount:   fileCount,
		FilesPruned: filesPruned,
	}

	if status == "completed" && resultData != nil {
		// Parse file scan tasks
		if tasks, ok := resultData["file_scan_tasks"].([]any); ok {
			for _, t := range tasks {
				if taskMap, ok := t.(map[string]any); ok {
					taskBytes, _ := json.Marshal(taskMap)
					var task FileScanTask
					if json.Unmarshal(taskBytes, &task) == nil {
						result.FileScanTasks = append(result.FileScanTasks, task)
					}
				}
			}
		}

		// Parse delete files
		if deletes, ok := resultData["delete_files"].([]any); ok {
			for _, d := range deletes {
				if delMap, ok := d.(map[string]any); ok {
					delBytes, _ := json.Marshal(delMap)
					var del DeleteFileInfo
					if json.Unmarshal(delBytes, &del) == nil {
						result.DeleteFiles = append(result.DeleteFiles, del)
					}
				}
			}
		}

		// Parse plan tasks
		if planTasks, ok := resultData["plan_tasks"].([]any); ok {
			for _, pt := range planTasks {
				if s, ok := pt.(string); ok {
					result.PlanTasks = append(result.PlanTasks, s)
				}
			}
		}
	}

	return result, nil
}

// FetchTasksForPlanTask resolves an opaque plan-task token to file-scan-tasks.
func (p *Planner) FetchTasksForPlanTask(ctx context.Context, planID, planTask string) ([]FileScanTask, []DeleteFileInfo, error) {
	result, err := p.GetPlanResult(ctx, planID)
	if err != nil {
		return nil, nil, err
	}

	if result.Status != "completed" {
		return nil, nil, fmt.Errorf("plan not ready: %s", result.Status)
	}

	// Decode plan task to get file indices
	indices, err := decodePlanTask(planTask)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid plan task: %w", err)
	}

	var tasks []FileScanTask
	for _, idx := range indices {
		if idx >= 0 && idx < len(result.FileScanTasks) {
			tasks = append(tasks, result.FileScanTasks[idx])
		}
	}

	return tasks, result.DeleteFiles, nil
}

// Helper methods

func (p *Planner) getTableMetadata(ctx context.Context, namespace []string, tableName string) (map[string]any, error) {
	var metadata map[string]any
	err := p.db.Pool.QueryRow(ctx, `
		SELECT t.metadata FROM tables t
		JOIN namespaces n ON t.namespace_id = n.id
		WHERE n.name = $1 AND t.name = $2
	`, namespace, tableName).Scan(&metadata)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("table not found: %s.%s", namespace, tableName)
		}
		return nil, fmt.Errorf("getting table metadata: %w", err)
	}

	return metadata, nil
}

func (p *Planner) resolveSnapshot(metadata map[string]any, requestedID *int64) (int64, map[string]any, error) {
	snapshots, _ := metadata["snapshots"].([]any)

	if len(snapshots) == 0 {
		return 0, nil, nil // Empty table
	}

	// If specific snapshot requested, find it
	if requestedID != nil {
		for _, s := range snapshots {
			snap, _ := s.(map[string]any)
			if snapID, ok := snap["snapshot-id"].(float64); ok && int64(snapID) == *requestedID {
				return int64(snapID), snap, nil
			}
		}
		return 0, nil, fmt.Errorf("snapshot not found: %d", *requestedID)
	}

	// Use current snapshot
	currentID, _ := metadata["current-snapshot-id"].(float64)
	for _, s := range snapshots {
		snap, _ := s.(map[string]any)
		if snapID, ok := snap["snapshot-id"].(float64); ok && int64(snapID) == int64(currentID) {
			return int64(currentID), snap, nil
		}
	}

	// Fall back to latest snapshot
	lastSnap, _ := snapshots[len(snapshots)-1].(map[string]any)
	snapID, _ := lastSnap["snapshot-id"].(float64)
	return int64(snapID), lastSnap, nil
}

func (p *Planner) readManifestList(ctx context.Context, path string) (*avro.ManifestList, error) {
	reader, err := p.storage.Read(ctx, path)
	if err != nil {
		return nil, err
	}
	return avro.ReadManifestList(ctx, reader)
}

func (p *Planner) readManifestsParallel(ctx context.Context, entries []avro.ManifestListEntry) []*avro.Manifest {
	if len(entries) == 0 {
		return nil
	}

	// Limit concurrency
	semaphore := make(chan struct{}, p.config.MaxConcurrentManifests)
	var wg sync.WaitGroup
	var mu sync.Mutex

	results := make([]*avro.Manifest, len(entries))

	for i, entry := range entries {
		wg.Add(1)
		go func(idx int, e avro.ManifestListEntry) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			reader, err := p.storage.Read(ctx, e.ManifestPath)
			if err != nil {
				slog.Error("failed to read manifest", "path", e.ManifestPath, "error", err)
				return
			}

			manifest, err := avro.ReadManifest(ctx, reader)
			if err != nil {
				slog.Error("failed to parse manifest", "path", e.ManifestPath, "error", err)
				return
			}

			mu.Lock()
			results[idx] = manifest
			mu.Unlock()
		}(i, entry)
	}

	wg.Wait()

	// Filter out nil results
	var manifests []*avro.Manifest
	for _, m := range results {
		if m != nil {
			manifests = append(manifests, m)
		}
	}

	return manifests
}

func (p *Planner) getSchema(schemas []any, schemaID int) []map[string]any {
	for _, s := range schemas {
		schema, _ := s.(map[string]any)
		if id, ok := schema["schema-id"].(float64); ok && int(id) == schemaID {
			if fields, ok := schema["fields"].([]any); ok {
				result := make([]map[string]any, 0, len(fields))
				for _, f := range fields {
					if field, ok := f.(map[string]any); ok {
						result = append(result, field)
					}
				}
				return result
			}
		}
	}
	return nil
}

func (p *Planner) buildFileScanTasks(files []avro.DataFile) []map[string]any {
	tasks := make([]map[string]any, len(files))
	for i, df := range files {
		info := NewDataFileInfo(&df, 0)
		infoBytes, _ := json.Marshal(info)
		var infoMap map[string]any
		json.Unmarshal(infoBytes, &infoMap)
		tasks[i] = map[string]any{
			"data-file": infoMap,
		}
	}
	return tasks
}

func (p *Planner) buildDeleteFiles(files []avro.DataFile) []map[string]any {
	result := make([]map[string]any, len(files))
	for i, df := range files {
		info := NewDeleteFileInfo(&df, 0)
		infoBytes, _ := json.Marshal(info)
		var infoMap map[string]any
		json.Unmarshal(infoBytes, &infoMap)
		result[i] = infoMap
	}
	return result
}

func (p *Planner) buildPlanTasks(files []avro.DataFile) []string {
	if len(files) <= p.config.PlanTaskBatchSize {
		// Single task for all files
		return []string{encodePlanTask(0, len(files))}
	}

	// Create batches
	var tasks []string
	for i := 0; i < len(files); i += p.config.PlanTaskBatchSize {
		end := i + p.config.PlanTaskBatchSize
		if end > len(files) {
			end = len(files)
		}
		tasks = append(tasks, encodePlanTask(i, end))
	}
	return tasks
}

func (p *Planner) failPlan(ctx context.Context, planID, errorMsg string) {
	_, _ = p.db.Pool.Exec(ctx, `
		UPDATE scan_plans
		SET status = 'failed', error_message = $2, completed_at = NOW()
		WHERE id = $1
	`, planID, errorMsg)
	metrics.RecordScanPlan("failed")
}

// Plan task encoding/decoding (simple base64 JSON for now)

func encodePlanTask(start, end int) string {
	data, _ := json.Marshal(map[string]int{"start": start, "end": end})
	return base64.StdEncoding.EncodeToString(data)
}

func decodePlanTask(task string) ([]int, error) {
	data, err := base64.StdEncoding.DecodeString(task)
	if err != nil {
		return nil, err
	}

	var bounds map[string]int
	if err := json.Unmarshal(data, &bounds); err != nil {
		return nil, err
	}

	start, end := bounds["start"], bounds["end"]
	indices := make([]int, 0, end-start)
	for i := start; i < end; i++ {
		indices = append(indices, i)
	}
	return indices, nil
}
