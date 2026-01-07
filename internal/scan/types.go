// Package scan provides server-side scan planning for Iceberg tables.
package scan

import (
	"github.com/kimuyb/bingsan/internal/avro"
)

// PlanRequest contains the parameters for a scan plan request.
type PlanRequest struct {
	TableID       string
	Namespace     []string
	TableName     string
	SnapshotID    *int64
	Filter        *Expression
	Select        []string
	CaseSensitive bool
}

// PlanResult contains the result of a scan plan operation.
type PlanResult struct {
	PlanID             string            `json:"plan-id"`
	Status             string            `json:"status"` // "submitted", "completed", "failed"
	FileScanTasks      []FileScanTask    `json:"file-scan-tasks,omitempty"`
	PlanTasks          []string          `json:"plan-tasks,omitempty"`
	DeleteFiles        []DeleteFileInfo  `json:"delete-files,omitempty"`
	StorageCredentials map[string]string `json:"storage-credentials,omitempty"`
	FileCount          int               `json:"-"`
	FilesPruned        int               `json:"-"`
}

// FileScanTask represents a unit of scan work for a single data file.
type FileScanTask struct {
	DataFile       DataFileInfo   `json:"data-file"`
	DeleteFileRefs []int          `json:"delete-file-references,omitempty"`
	ResidualFilter map[string]any `json:"residual-filter,omitempty"`
}

// DataFileInfo contains file metadata matching the OpenAPI spec.
type DataFileInfo struct {
	Content         string         `json:"content"`
	FilePath        string         `json:"file-path"`
	FileFormat      string         `json:"file-format"`
	SpecID          int            `json:"spec-id"`
	Partition       map[string]any `json:"partition,omitempty"`
	FileSizeBytes   int64          `json:"file-size-in-bytes"`
	RecordCount     int64          `json:"record-count"`
	ColumnSizes     map[int]int64  `json:"column-sizes,omitempty"`
	ValueCounts     map[int]int64  `json:"value-counts,omitempty"`
	NullValueCounts map[int]int64  `json:"null-value-counts,omitempty"`
	NaNValueCounts  map[int]int64  `json:"nan-value-counts,omitempty"`
	LowerBounds     map[int]any    `json:"lower-bounds,omitempty"`
	UpperBounds     map[int]any    `json:"upper-bounds,omitempty"`
	SplitOffsets    []int64        `json:"split-offsets,omitempty"`
	SortOrderID     *int           `json:"sort-order-id,omitempty"`
}

// DeleteFileInfo contains delete file metadata.
type DeleteFileInfo struct {
	Content       string         `json:"content"` // "position-deletes" or "equality-deletes"
	FilePath      string         `json:"file-path"`
	FileFormat    string         `json:"file-format"`
	SpecID        int            `json:"spec-id"`
	Partition     map[string]any `json:"partition,omitempty"`
	FileSizeBytes int64          `json:"file-size-in-bytes"`
	RecordCount   int64          `json:"record-count"`
	EqualityIDs   []int          `json:"equality-ids,omitempty"`
}

// NewDataFileInfo converts an avro.DataFile to DataFileInfo for API response.
func NewDataFileInfo(df *avro.DataFile, specID int) DataFileInfo {
	info := DataFileInfo{
		Content:       "data",
		FilePath:      df.FilePath,
		FileFormat:    df.FileFormatString(),
		SpecID:        specID,
		Partition:     df.Partition,
		FileSizeBytes: df.FileSizeBytes,
		RecordCount:   df.RecordCount,
	}

	// Convert column stats maps (int32 -> int for JSON)
	if len(df.ColumnSizes) > 0 {
		info.ColumnSizes = convertIntMap(df.ColumnSizes)
	}
	if len(df.ValueCounts) > 0 {
		info.ValueCounts = convertIntMap(df.ValueCounts)
	}
	if len(df.NullValueCounts) > 0 {
		info.NullValueCounts = convertIntMap(df.NullValueCounts)
	}
	if len(df.NaNValueCounts) > 0 {
		info.NaNValueCounts = convertIntMap(df.NaNValueCounts)
	}

	// Convert bounds (keeping as bytes for now, could decode based on schema)
	if len(df.LowerBounds) > 0 {
		info.LowerBounds = convertBoundsMap(df.LowerBounds)
	}
	if len(df.UpperBounds) > 0 {
		info.UpperBounds = convertBoundsMap(df.UpperBounds)
	}

	if len(df.SplitOffsets) > 0 {
		info.SplitOffsets = df.SplitOffsets
	}
	if df.SortOrderID != nil {
		sortOrderID := int(*df.SortOrderID)
		info.SortOrderID = &sortOrderID
	}

	return info
}

// NewDeleteFileInfo converts an avro.DataFile (delete type) to DeleteFileInfo.
func NewDeleteFileInfo(df *avro.DataFile, specID int) DeleteFileInfo {
	content := "position-deletes"
	if df.Content == avro.ContentEqualityDeletes {
		content = "equality-deletes"
	}

	info := DeleteFileInfo{
		Content:       content,
		FilePath:      df.FilePath,
		FileFormat:    df.FileFormatString(),
		SpecID:        specID,
		Partition:     df.Partition,
		FileSizeBytes: df.FileSizeBytes,
		RecordCount:   df.RecordCount,
	}

	if len(df.EqualityIDs) > 0 {
		info.EqualityIDs = make([]int, len(df.EqualityIDs))
		for i, id := range df.EqualityIDs {
			info.EqualityIDs[i] = int(id)
		}
	}

	return info
}

// convertIntMap converts map[int32]int64 to map[int]int64.
func convertIntMap(m map[int32]int64) map[int]int64 {
	result := make(map[int]int64, len(m))
	for k, v := range m {
		result[int(k)] = v
	}
	return result
}

// convertBoundsMap converts map[int32][]byte to map[int]any.
// For now, we keep bounds as base64-encoded bytes.
// A future improvement could decode based on schema types.
func convertBoundsMap(m map[int32][]byte) map[int]any {
	result := make(map[int]any, len(m))
	for k, v := range m {
		result[int(k)] = v
	}
	return result
}
