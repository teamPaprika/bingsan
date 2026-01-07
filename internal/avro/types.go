// Package avro provides Iceberg manifest file parsing using the Avro format.
package avro

// ManifestListEntry represents an entry in an Iceberg manifest list file.
// The manifest list contains references to all manifest files for a snapshot.
type ManifestListEntry struct {
	ManifestPath       string         `avro:"manifest_path"`
	ManifestLength     int64          `avro:"manifest_length"`
	PartitionSpecID    int32          `avro:"partition_spec_id"`
	Content            int32          `avro:"content"` // 0=data, 1=deletes
	SequenceNumber     int64          `avro:"sequence_number"`
	MinSequenceNumber  int64          `avro:"min_sequence_number"`
	AddedSnapshotID    int64          `avro:"added_snapshot_id"`
	AddedFilesCount    *int32         `avro:"added_files_count"`
	ExistingFilesCount *int32         `avro:"existing_files_count"`
	DeletedFilesCount  *int32         `avro:"deleted_files_count"`
	AddedRowsCount     *int64         `avro:"added_rows_count"`
	ExistingRowsCount  *int64         `avro:"existing_rows_count"`
	DeletedRowsCount   *int64         `avro:"deleted_rows_count"`
	Partitions         []FieldSummary `avro:"partitions"`
}

// FieldSummary contains statistics for a partition field.
type FieldSummary struct {
	ContainsNull bool   `avro:"contains_null"`
	ContainsNaN  *bool  `avro:"contains_nan"`
	LowerBound   []byte `avro:"lower_bound"`
	UpperBound   []byte `avro:"upper_bound"`
}

// ManifestEntry represents an entry in an Iceberg manifest file.
// Each entry describes a data file or delete file.
type ManifestEntry struct {
	Status             int32    `avro:"status"` // 0=existing, 1=added, 2=deleted
	SnapshotID         *int64   `avro:"snapshot_id"`
	SequenceNumber     *int64   `avro:"sequence_number"`
	FileSequenceNumber *int64   `avro:"file_sequence_number"`
	DataFile           DataFile `avro:"data_file"`
}

// ManifestEntryStatus constants.
const (
	StatusExisting = 0
	StatusAdded    = 1
	StatusDeleted  = 2
)

// DataFile represents file-level metadata in an Iceberg manifest.
type DataFile struct {
	Content         int32              `avro:"content"` // 0=data, 1=position_deletes, 2=equality_deletes
	FilePath        string             `avro:"file_path"`
	FileFormat      string             `avro:"file_format"`
	Partition       map[string]any     `avro:"partition"`
	RecordCount     int64              `avro:"record_count"`
	FileSizeBytes   int64              `avro:"file_size_in_bytes"`
	ColumnSizes     map[int32]int64    `avro:"column_sizes"`
	ValueCounts     map[int32]int64    `avro:"value_counts"`
	NullValueCounts map[int32]int64    `avro:"null_value_counts"`
	NaNValueCounts  map[int32]int64    `avro:"nan_value_counts"`
	LowerBounds     map[int32][]byte   `avro:"lower_bounds"`
	UpperBounds     map[int32][]byte   `avro:"upper_bounds"`
	KeyMetadata     []byte             `avro:"key_metadata"`
	SplitOffsets    []int64            `avro:"split_offsets"`
	EqualityIDs     []int32            `avro:"equality_ids"`
	SortOrderID     *int32             `avro:"sort_order_id"`
}

// DataFileContent constants.
const (
	ContentData            = 0
	ContentPositionDeletes = 1
	ContentEqualityDeletes = 2
)

// FileFormat returns the file format as a lowercase string.
func (d *DataFile) FileFormatString() string {
	switch d.FileFormat {
	case "PARQUET", "parquet":
		return "parquet"
	case "ORC", "orc":
		return "orc"
	case "AVRO", "avro":
		return "avro"
	default:
		return d.FileFormat
	}
}

// IsDataFile returns true if this is a data file (not a delete file).
func (d *DataFile) IsDataFile() bool {
	return d.Content == ContentData
}

// IsDeleteFile returns true if this is a delete file.
func (d *DataFile) IsDeleteFile() bool {
	return d.Content == ContentPositionDeletes || d.Content == ContentEqualityDeletes
}
