package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kimuyb/bingsan/internal/api/handlers/updates"
)

// =============================================================================
// Update Processor Tests
// =============================================================================

func TestProcessorNewCreatesDeepCopy(t *testing.T) {
	original := map[string]any{
		"table-uuid":  "original-uuid",
		"snapshots":   []any{},
		"schemas":     []any{map[string]any{"schema-id": 0}},
		"refs":        map[string]any{},
	}

	processor := updates.NewProcessor(original)
	result := processor.Result()

	// Modify result
	result["table-uuid"] = "modified-uuid"

	// Original should be unchanged
	assert.Equal(t, "original-uuid", original["table-uuid"])
}

func TestProcessorApplyAddSnapshot(t *testing.T) {
	metadata := map[string]any{
		"table-uuid":           "test-uuid",
		"snapshots":            []any{},
		"refs":                 map[string]any{},
		"last-sequence-number": int64(0),
		"snapshot-log":         []any{},
	}

	rawUpdates := []updates.RawUpdate{
		{
			Action: "add-snapshot",
			Raw:    []byte(`{"snapshot":{"snapshot-id":123456789,"sequence-number":1,"timestamp-ms":1700000000000,"summary":{"operation":"append"}}}`),
		},
	}

	processor := updates.NewProcessor(metadata)
	err := processor.Apply(rawUpdates)
	require.NoError(t, err)

	result := processor.Result()
	snapshots := result["snapshots"].([]any)
	assert.Len(t, snapshots, 1)

	snapshot := snapshots[0].(map[string]any)
	assert.Equal(t, float64(123456789), snapshot["snapshot-id"])
}

func TestProcessorApplySetSnapshotRef(t *testing.T) {
	metadata := map[string]any{
		"table-uuid": "test-uuid",
		"snapshots": []any{
			map[string]any{"snapshot-id": int64(123456789)},
		},
		"refs": map[string]any{},
	}

	rawUpdates := []updates.RawUpdate{
		{
			Action: "set-snapshot-ref",
			Raw:    []byte(`{"ref-name":"main","type":"branch","snapshot-id":123456789}`),
		},
	}

	processor := updates.NewProcessor(metadata)
	err := processor.Apply(rawUpdates)
	require.NoError(t, err)

	result := processor.Result()
	refs := result["refs"].(map[string]any)
	mainRef := refs["main"].(map[string]any)

	assert.Equal(t, "branch", mainRef["type"])
	assert.Equal(t, int64(123456789), mainRef["snapshot-id"])
	assert.Equal(t, int64(123456789), result["current-snapshot-id"])
}

func TestProcessorApplyAddSchema(t *testing.T) {
	metadata := map[string]any{
		"table-uuid": "test-uuid",
		"schemas": []any{
			map[string]any{"schema-id": 0, "fields": []any{}},
		},
	}

	rawUpdates := []updates.RawUpdate{
		{
			Action: "add-schema",
			Raw:    []byte(`{"schema":{"schema-id":1,"fields":[{"id":1,"name":"id","type":"string"}]}}`),
		},
	}

	processor := updates.NewProcessor(metadata)
	err := processor.Apply(rawUpdates)
	require.NoError(t, err)

	result := processor.Result()
	schemas := result["schemas"].([]any)
	assert.Len(t, schemas, 2)
}

func TestProcessorApplySetCurrentSchema(t *testing.T) {
	metadata := map[string]any{
		"table-uuid":        "test-uuid",
		"current-schema-id": 0,
		"schemas": []any{
			map[string]any{"schema-id": 0},
			map[string]any{"schema-id": 1},
		},
	}

	rawUpdates := []updates.RawUpdate{
		{
			Action: "set-current-schema",
			Raw:    []byte(`{"schema-id":1}`),
		},
	}

	processor := updates.NewProcessor(metadata)
	err := processor.Apply(rawUpdates)
	require.NoError(t, err)

	result := processor.Result()
	assert.Equal(t, 1, result["current-schema-id"])
}

func TestProcessorApplyAddPartitionSpec(t *testing.T) {
	metadata := map[string]any{
		"table-uuid": "test-uuid",
		"partition-specs": []any{
			map[string]any{"spec-id": 0, "fields": []any{}},
		},
	}

	rawUpdates := []updates.RawUpdate{
		{
			Action: "add-spec",
			Raw:    []byte(`{"spec":{"spec-id":1,"fields":[{"source-id":1,"transform":"identity","name":"id"}]}}`),
		},
	}

	processor := updates.NewProcessor(metadata)
	err := processor.Apply(rawUpdates)
	require.NoError(t, err)

	result := processor.Result()
	specs := result["partition-specs"].([]any)
	assert.Len(t, specs, 2)
}

func TestProcessorApplySetDefaultPartitionSpec(t *testing.T) {
	metadata := map[string]any{
		"table-uuid":      "test-uuid",
		"default-spec-id": 0,
		"partition-specs": []any{
			map[string]any{"spec-id": 0},
			map[string]any{"spec-id": 1},
		},
	}

	rawUpdates := []updates.RawUpdate{
		{
			Action: "set-default-spec",
			Raw:    []byte(`{"spec-id":1}`),
		},
	}

	processor := updates.NewProcessor(metadata)
	err := processor.Apply(rawUpdates)
	require.NoError(t, err)

	result := processor.Result()
	assert.Equal(t, 1, result["default-spec-id"])
}

func TestProcessorApplyAddSortOrder(t *testing.T) {
	metadata := map[string]any{
		"table-uuid": "test-uuid",
		"sort-orders": []any{
			map[string]any{"order-id": 0, "fields": []any{}},
		},
	}

	rawUpdates := []updates.RawUpdate{
		{
			Action: "add-sort-order",
			Raw:    []byte(`{"sort-order":{"order-id":1,"fields":[{"source-id":1,"direction":"asc"}]}}`),
		},
	}

	processor := updates.NewProcessor(metadata)
	err := processor.Apply(rawUpdates)
	require.NoError(t, err)

	result := processor.Result()
	orders := result["sort-orders"].([]any)
	assert.Len(t, orders, 2)
}

func TestProcessorApplySetProperties(t *testing.T) {
	metadata := map[string]any{
		"table-uuid": "test-uuid",
		"properties": map[string]any{
			"existing": "value",
		},
	}

	rawUpdates := []updates.RawUpdate{
		{
			Action: "set-properties",
			Raw:    []byte(`{"updates":{"new-key":"new-value","another":"prop"}}`),
		},
	}

	processor := updates.NewProcessor(metadata)
	err := processor.Apply(rawUpdates)
	require.NoError(t, err)

	result := processor.Result()
	props := result["properties"].(map[string]any)
	assert.Equal(t, "value", props["existing"])
	assert.Equal(t, "new-value", props["new-key"])
	assert.Equal(t, "prop", props["another"])
}

func TestProcessorApplyRemoveProperties(t *testing.T) {
	metadata := map[string]any{
		"table-uuid": "test-uuid",
		"properties": map[string]any{
			"keep":   "value",
			"remove": "this",
		},
	}

	rawUpdates := []updates.RawUpdate{
		{
			Action: "remove-properties",
			Raw:    []byte(`{"removals":["remove"]}`),
		},
	}

	processor := updates.NewProcessor(metadata)
	err := processor.Apply(rawUpdates)
	require.NoError(t, err)

	result := processor.Result()
	props := result["properties"].(map[string]any)
	assert.Equal(t, "value", props["keep"])
	_, exists := props["remove"]
	assert.False(t, exists)
}

func TestProcessorApplySetLocation(t *testing.T) {
	metadata := map[string]any{
		"table-uuid": "test-uuid",
		"location":   "s3://old-bucket/path",
	}

	rawUpdates := []updates.RawUpdate{
		{
			Action: "set-location",
			Raw:    []byte(`{"location":"s3://new-bucket/new-path"}`),
		},
	}

	processor := updates.NewProcessor(metadata)
	err := processor.Apply(rawUpdates)
	require.NoError(t, err)

	result := processor.Result()
	assert.Equal(t, "s3://new-bucket/new-path", result["location"])
}

func TestProcessorApplyAssignUUID(t *testing.T) {
	metadata := map[string]any{
		"table-uuid": "",
	}

	rawUpdates := []updates.RawUpdate{
		{
			Action: "assign-uuid",
			Raw:    []byte(`{"uuid":"new-uuid-value"}`),
		},
	}

	processor := updates.NewProcessor(metadata)
	err := processor.Apply(rawUpdates)
	require.NoError(t, err)

	result := processor.Result()
	assert.Equal(t, "new-uuid-value", result["table-uuid"])
}

func TestProcessorApplyUpgradeFormatVersion(t *testing.T) {
	metadata := map[string]any{
		"format-version": 1,
	}

	rawUpdates := []updates.RawUpdate{
		{
			Action: "upgrade-format-version",
			Raw:    []byte(`{"format-version":2}`),
		},
	}

	processor := updates.NewProcessor(metadata)
	err := processor.Apply(rawUpdates)
	require.NoError(t, err)

	result := processor.Result()
	assert.Equal(t, 2, result["format-version"])
}

func TestProcessorProcessingOrder(t *testing.T) {
	// Test that updates are processed in the correct dependency order
	// Even if provided out of order, schemas should be added before snapshots
	metadata := map[string]any{
		"table-uuid":           "test-uuid",
		"schemas":              []any{map[string]any{"schema-id": 0}},
		"snapshots":            []any{},
		"refs":                 map[string]any{},
		"last-sequence-number": int64(0),
		"snapshot-log":         []any{},
	}

	// Provide updates in wrong order - snapshot before schema
	rawUpdates := []updates.RawUpdate{
		{
			Action: "add-snapshot",
			Raw:    []byte(`{"snapshot":{"snapshot-id":123,"sequence-number":1,"timestamp-ms":1700000000000}}`),
		},
		{
			Action: "add-schema",
			Raw:    []byte(`{"schema":{"schema-id":1,"fields":[]}}`),
		},
	}

	processor := updates.NewProcessor(metadata)
	err := processor.Apply(rawUpdates)
	require.NoError(t, err)

	result := processor.Result()
	schemas := result["schemas"].([]any)
	snapshots := result["snapshots"].([]any)

	// Both should be added successfully
	assert.Len(t, schemas, 2)
	assert.Len(t, snapshots, 1)
}

func TestProcessorApplyMultipleUpdatesOfSameType(t *testing.T) {
	metadata := map[string]any{
		"table-uuid":           "test-uuid",
		"snapshots":            []any{},
		"refs":                 map[string]any{},
		"last-sequence-number": int64(0),
		"snapshot-log":         []any{},
	}

	rawUpdates := []updates.RawUpdate{
		{
			Action: "add-snapshot",
			Raw:    []byte(`{"snapshot":{"snapshot-id":1,"sequence-number":1,"timestamp-ms":1700000000001}}`),
		},
		{
			Action: "add-snapshot",
			Raw:    []byte(`{"snapshot":{"snapshot-id":2,"sequence-number":2,"timestamp-ms":1700000000002}}`),
		},
		{
			Action: "add-snapshot",
			Raw:    []byte(`{"snapshot":{"snapshot-id":3,"sequence-number":3,"timestamp-ms":1700000000003}}`),
		},
	}

	processor := updates.NewProcessor(metadata)
	err := processor.Apply(rawUpdates)
	require.NoError(t, err)

	result := processor.Result()
	snapshots := result["snapshots"].([]any)
	assert.Len(t, snapshots, 3)
}

func TestProcessorApplyEmptyUpdates(t *testing.T) {
	metadata := map[string]any{
		"table-uuid": "test-uuid",
		"snapshots":  []any{},
	}

	rawUpdates := []updates.RawUpdate{}

	processor := updates.NewProcessor(metadata)
	err := processor.Apply(rawUpdates)
	require.NoError(t, err)

	result := processor.Result()
	assert.Equal(t, "test-uuid", result["table-uuid"])
}

func TestProcessorApplyUnknownActionReturnsError(t *testing.T) {
	metadata := map[string]any{
		"table-uuid": "test-uuid",
	}

	rawUpdates := []updates.RawUpdate{
		{
			Action: "unknown-action-type",
			Raw:    []byte(`{"foo":"bar"}`),
		},
	}

	processor := updates.NewProcessor(metadata)
	err := processor.Apply(rawUpdates)
	// Unknown actions should return an error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action")
}

func TestProcessorApplyInvalidJSON(t *testing.T) {
	metadata := map[string]any{
		"table-uuid": "test-uuid",
		"snapshots":  []any{},
	}

	rawUpdates := []updates.RawUpdate{
		{
			Action: "add-snapshot",
			Raw:    []byte(`{invalid json`),
		},
	}

	processor := updates.NewProcessor(metadata)
	err := processor.Apply(rawUpdates)
	// Should return an error for invalid JSON
	require.Error(t, err)
}

// =============================================================================
// RawUpdate Unmarshal Tests
// =============================================================================

func TestRawUpdateUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name           string
		json           string
		expectedAction string
		expectError    bool
	}{
		{
			name:           "add-snapshot action",
			json:           `{"action":"add-snapshot","snapshot":{"snapshot-id":123}}`,
			expectedAction: "add-snapshot",
			expectError:    false,
		},
		{
			name:           "set-snapshot-ref action",
			json:           `{"action":"set-snapshot-ref","ref-name":"main","type":"branch"}`,
			expectedAction: "set-snapshot-ref",
			expectError:    false,
		},
		{
			name:           "missing action field",
			json:           `{"snapshot":{"snapshot-id":123}}`,
			expectedAction: "",
			expectError:    false,
		},
		{
			name:        "invalid json",
			json:        `{invalid`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var update updates.RawUpdate
			err := update.UnmarshalJSON([]byte(tt.json))

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedAction, update.Action)
			}
		})
	}
}

// =============================================================================
// Update Error Tests
// =============================================================================

func TestUpdateErrorMessage(t *testing.T) {
	err := &updates.UpdateError{
		Action:  "add-snapshot",
		Message: "invalid snapshot data",
	}

	assert.Contains(t, err.Error(), "add-snapshot")
	assert.Contains(t, err.Error(), "invalid snapshot data")
}

func TestUpdateErrorWithCause(t *testing.T) {
	cause := assert.AnError
	err := &updates.UpdateError{
		Action:  "set-properties",
		Message: "failed to update",
		Cause:   cause,
	}

	assert.Contains(t, err.Error(), "set-properties")
	assert.Equal(t, cause, err.Unwrap())
}
