// Package updates provides table metadata update processing for the Iceberg REST Catalog.
package updates

import "encoding/json"

// RawUpdate represents an unparsed update from the CommitTable request.
// It captures the action type and preserves the raw JSON for type-specific parsing.
type RawUpdate struct {
	Action string          `json:"action"`
	Raw    json.RawMessage `json:"-"`
}

// UnmarshalJSON implements custom unmarshaling to capture both action and raw data.
func (u *RawUpdate) UnmarshalJSON(data []byte) error {
	// First extract just the action
	var action struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal(data, &action); err != nil {
		return err
	}
	u.Action = action.Action
	u.Raw = data
	return nil
}

// ============================================================================
// Core Table Operations
// ============================================================================

// AssignUUID assigns a UUID to a table (typically during creation).
type AssignUUID struct {
	UUID string `json:"uuid"`
}

// UpgradeFormatVersion upgrades the table format version.
type UpgradeFormatVersion struct {
	FormatVersion int `json:"format-version"`
}

// SetLocation changes the table's base location.
type SetLocation struct {
	Location string `json:"location"`
}

// SetProperties sets table properties (key-value pairs).
type SetProperties struct {
	Updates map[string]string `json:"updates"`
}

// RemoveProperties removes table properties by key.
type RemoveProperties struct {
	Removals []string `json:"removals"`
}

// ============================================================================
// Schema Management
// ============================================================================

// AddSchema adds a new schema version to the table.
type AddSchema struct {
	Schema       map[string]any `json:"schema"`
	LastColumnID *int           `json:"last-column-id,omitempty"` // Deprecated
}

// SetCurrentSchema sets the current schema by ID (-1 means last added).
type SetCurrentSchema struct {
	SchemaID int `json:"schema-id"`
}

// RemoveSchemas removes schema versions by ID.
type RemoveSchemas struct {
	SchemaIDs []int `json:"schema-ids"`
}

// ============================================================================
// Partition Spec Management
// ============================================================================

// AddSpec adds a new partition specification.
type AddSpec struct {
	Spec map[string]any `json:"spec"`
}

// SetDefaultSpec sets the default partition spec (-1 means last added).
type SetDefaultSpec struct {
	SpecID int `json:"spec-id"`
}

// RemovePartitionSpecs removes partition specs by ID.
type RemovePartitionSpecs struct {
	SpecIDs []int `json:"spec-ids"`
}

// ============================================================================
// Sort Order Management
// ============================================================================

// AddSortOrder adds a new sort order.
type AddSortOrder struct {
	SortOrder map[string]any `json:"sort-order"`
}

// SetDefaultSortOrder sets the default sort order (-1 means last added).
type SetDefaultSortOrder struct {
	SortOrderID int `json:"sort-order-id"`
}

// ============================================================================
// Snapshot Management (Critical for Data Operations)
// ============================================================================

// AddSnapshot adds a new snapshot to the table.
type AddSnapshot struct {
	Snapshot map[string]any `json:"snapshot"`
}

// SetSnapshotRef creates or updates a snapshot reference (branch or tag).
type SetSnapshotRef struct {
	RefName            string `json:"ref-name"`
	Type               string `json:"type"`        // "branch" or "tag"
	SnapshotID         int64  `json:"snapshot-id"`
	MaxRefAgeMs        *int64 `json:"max-ref-age-ms,omitempty"`
	MaxSnapshotAgeMs   *int64 `json:"max-snapshot-age-ms,omitempty"`
	MinSnapshotsToKeep *int   `json:"min-snapshots-to-keep,omitempty"`
}

// RemoveSnapshots removes snapshots by ID.
type RemoveSnapshots struct {
	SnapshotIDs []int64 `json:"snapshot-ids"`
}

// RemoveSnapshotRef removes a snapshot reference by name.
type RemoveSnapshotRef struct {
	RefName string `json:"ref-name"`
}

// ============================================================================
// Statistics Management
// ============================================================================

// SetStatistics sets table statistics for a snapshot.
type SetStatistics struct {
	SnapshotID *int64         `json:"snapshot-id,omitempty"` // Deprecated
	Statistics map[string]any `json:"statistics"`
}

// RemoveStatistics removes statistics for a snapshot.
type RemoveStatistics struct {
	SnapshotID int64 `json:"snapshot-id"`
}

// SetPartitionStatistics sets partition-level statistics.
type SetPartitionStatistics struct {
	PartitionStatistics map[string]any `json:"partition-statistics"`
}

// RemovePartitionStatistics removes partition statistics for a snapshot.
type RemovePartitionStatistics struct {
	SnapshotID int64 `json:"snapshot-id"`
}

// ============================================================================
// Encryption (Format Version 3)
// ============================================================================

// AddEncryptionKey adds an encryption key to the table.
type AddEncryptionKey struct {
	EncryptionKey map[string]any `json:"encryption-key"`
}

// RemoveEncryptionKey removes an encryption key by ID.
type RemoveEncryptionKey struct {
	KeyID string `json:"key-id"`
}

// ============================================================================
// View-Specific (for completeness, handled separately)
// ============================================================================

// AddViewVersion adds a new view version.
type AddViewVersion struct {
	ViewVersion map[string]any `json:"view-version"`
}

// SetCurrentViewVersion sets the current view version.
type SetCurrentViewVersion struct {
	ViewVersionID int `json:"view-version-id"`
}
