package updates

import (
	"encoding/json"
	"fmt"
)

// processingOrder defines the dependency-correct order for applying updates.
// Updates are applied in this sequence to ensure referential integrity:
// core table attributes, schema additions, partition spec additions,
// sort order additions, snapshot additions, snapshot references,
// statistics, properties, and removals (in reverse dependency order).
var processingOrder = []string{
	// Core operations
	"assign-uuid",
	"upgrade-format-version",
	"set-location",

	// Schema management (add before set)
	"add-schema",
	"set-current-schema",

	// Partition management (add before set)
	"add-spec",
	"set-default-spec",

	// Sort order management (add before set)
	"add-sort-order",
	"set-default-sort-order",

	// Snapshot management (critical for writes)
	"add-snapshot",
	"set-snapshot-ref",

	// Statistics (reference snapshots)
	"set-statistics",
	"set-partition-statistics",

	// Properties
	"set-properties",
	"remove-properties",

	// Cleanup operations (reverse dependency order)
	"remove-statistics",
	"remove-partition-statistics",
	"remove-snapshot-ref",
	"remove-snapshots",
	"remove-partition-specs",
	"remove-schemas",

	// Encryption
	"add-encryption-key",
	"remove-encryption-key",
}

// Processor handles applying updates to table metadata in dependency order.
type Processor struct {
	metadata map[string]any
}

// NewProcessor creates a processor with a deep copy of metadata.
// The copy ensures that failures don't corrupt the original metadata.
func NewProcessor(metadata map[string]any) *Processor {
	return &Processor{
		metadata: deepCopyMap(metadata),
	}
}

// Apply processes all updates in dependency order.
// It parses, validates, and applies updates sequentially.
// Returns an error if any update fails; the metadata remains unchanged on error.
func (p *Processor) Apply(rawUpdates []RawUpdate) error {
	if len(rawUpdates) == 0 {
		return nil
	}

	// Phase 1: Parse all updates into typed structures
	parsed, err := p.parseUpdates(rawUpdates)
	if err != nil {
		return err
	}

	// Phase 2: Apply updates in dependency order
	for _, action := range processingOrder {
		updates, ok := parsed[action]
		if !ok {
			continue
		}

		for _, u := range updates {
			if err := p.applyUpdate(action, u); err != nil {
				return &UpdateError{
					Action:  action,
					Message: err.Error(),
				}
			}
		}
	}

	// Check for unknown actions
	for action := range parsed {
		if !isKnownAction(action) {
			return &UpdateError{
				Action:  action,
				Message: "unknown update action",
			}
		}
	}

	return nil
}

// Result returns the modified metadata after all updates are applied.
func (p *Processor) Result() map[string]any {
	return p.metadata
}

// parseUpdates parses raw updates into typed structures grouped by action.
func (p *Processor) parseUpdates(rawUpdates []RawUpdate) (map[string][]any, error) {
	parsed := make(map[string][]any)

	for i, raw := range rawUpdates {
		typed, err := parseUpdate(raw)
		if err != nil {
			return nil, fmt.Errorf("update[%d] (%s): %w", i, raw.Action, err)
		}
		parsed[raw.Action] = append(parsed[raw.Action], typed)
	}

	return parsed, nil
}

// parseUpdate parses a single raw update into its typed structure.
//
//nolint:gocyclo // Switch statement is inherently complex but clear
func parseUpdate(raw RawUpdate) (any, error) {
	var target any

	switch raw.Action {
	// Core operations
	case "assign-uuid":
		target = &AssignUUID{}
	case "upgrade-format-version":
		target = &UpgradeFormatVersion{}
	case "set-location":
		target = &SetLocation{}
	case "set-properties":
		target = &SetProperties{}
	case "remove-properties":
		target = &RemoveProperties{}

	// Schema management
	case "add-schema":
		target = &AddSchema{}
	case "set-current-schema":
		target = &SetCurrentSchema{}
	case "remove-schemas":
		target = &RemoveSchemas{}

	// Partition management
	case "add-spec":
		target = &AddSpec{}
	case "set-default-spec":
		target = &SetDefaultSpec{}
	case "remove-partition-specs":
		target = &RemovePartitionSpecs{}

	// Sort order management
	case "add-sort-order":
		target = &AddSortOrder{}
	case "set-default-sort-order":
		target = &SetDefaultSortOrder{}

	// Snapshot management
	case "add-snapshot":
		target = &AddSnapshot{}
	case "set-snapshot-ref":
		target = &SetSnapshotRef{}
	case "remove-snapshots":
		target = &RemoveSnapshots{}
	case "remove-snapshot-ref":
		target = &RemoveSnapshotRef{}

	// Statistics
	case "set-statistics":
		target = &SetStatistics{}
	case "remove-statistics":
		target = &RemoveStatistics{}
	case "set-partition-statistics":
		target = &SetPartitionStatistics{}
	case "remove-partition-statistics":
		target = &RemovePartitionStatistics{}

	// Encryption
	case "add-encryption-key":
		target = &AddEncryptionKey{}
	case "remove-encryption-key":
		target = &RemoveEncryptionKey{}

	default:
		return nil, fmt.Errorf("unknown action: %s", raw.Action)
	}

	if err := json.Unmarshal(raw.Raw, target); err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

	return target, nil
}

// applyUpdate dispatches to the appropriate handler based on action type.
// Type assertions are safe here because parseUpdate guarantees correct types.
//
//nolint:gocyclo,errcheck // Switch is inherently complex; type assertions safe per parseUpdate
func (p *Processor) applyUpdate(action string, data any) error {
	switch action {
	// Core operations
	case "assign-uuid":
		return p.applyAssignUUID(data.(*AssignUUID))
	case "upgrade-format-version":
		return p.applyUpgradeFormatVersion(data.(*UpgradeFormatVersion))
	case "set-location":
		return p.applySetLocation(data.(*SetLocation))
	case "set-properties":
		return p.applySetProperties(data.(*SetProperties))
	case "remove-properties":
		return p.applyRemoveProperties(data.(*RemoveProperties))

	// Schema management
	case "add-schema":
		return p.applyAddSchema(data.(*AddSchema))
	case "set-current-schema":
		return p.applySetCurrentSchema(data.(*SetCurrentSchema))
	case "remove-schemas":
		return p.applyRemoveSchemas(data.(*RemoveSchemas))

	// Partition management
	case "add-spec":
		return p.applyAddSpec(data.(*AddSpec))
	case "set-default-spec":
		return p.applySetDefaultSpec(data.(*SetDefaultSpec))
	case "remove-partition-specs":
		return p.applyRemovePartitionSpecs(data.(*RemovePartitionSpecs))

	// Sort order management
	case "add-sort-order":
		return p.applyAddSortOrder(data.(*AddSortOrder))
	case "set-default-sort-order":
		return p.applySetDefaultSortOrder(data.(*SetDefaultSortOrder))

	// Snapshot management
	case "add-snapshot":
		return p.applyAddSnapshot(data.(*AddSnapshot))
	case "set-snapshot-ref":
		return p.applySetSnapshotRef(data.(*SetSnapshotRef))
	case "remove-snapshots":
		return p.applyRemoveSnapshots(data.(*RemoveSnapshots))
	case "remove-snapshot-ref":
		return p.applyRemoveSnapshotRef(data.(*RemoveSnapshotRef))

	// Statistics
	case "set-statistics":
		return p.applySetStatistics(data.(*SetStatistics))
	case "remove-statistics":
		return p.applyRemoveStatistics(data.(*RemoveStatistics))
	case "set-partition-statistics":
		return p.applySetPartitionStatistics(data.(*SetPartitionStatistics))
	case "remove-partition-statistics":
		return p.applyRemovePartitionStatistics(data.(*RemovePartitionStatistics))

	// Encryption
	case "add-encryption-key":
		return p.applyAddEncryptionKey(data.(*AddEncryptionKey))
	case "remove-encryption-key":
		return p.applyRemoveEncryptionKey(data.(*RemoveEncryptionKey))

	default:
		return fmt.Errorf("unhandled action: %s", action)
	}
}

// isKnownAction checks if an action is in the processing order.
func isKnownAction(action string) bool {
	for _, a := range processingOrder {
		if a == action {
			return true
		}
	}
	return false
}
