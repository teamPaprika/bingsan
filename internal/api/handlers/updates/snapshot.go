package updates

import "fmt"

// Ref type constants.
const (
	refTypeBranch = "branch"
	refTypeTag    = "tag"
)

// ============================================================================
// Snapshot Management - Critical for Data Operations
// ============================================================================

// applyAddSnapshot adds a new snapshot to the table metadata.
// This is the core operation for data writes (append, overwrite).
func (p *Processor) applyAddSnapshot(u *AddSnapshot) error {
	if u.Snapshot == nil {
		return fmt.Errorf("snapshot is required")
	}

	// Validate required snapshot fields
	snapshotID, ok := getInt64(u.Snapshot, "snapshot-id")
	if !ok {
		return fmt.Errorf("snapshot-id is required")
	}

	if _, ok := getInt64(u.Snapshot, "timestamp-ms"); !ok {
		return fmt.Errorf("timestamp-ms is required")
	}

	// Get or initialize snapshots array
	snapshots := ensureSlice(p.metadata, "snapshots")

	// Check for duplicate snapshot ID
	for _, s := range snapshots {
		if snap, ok := s.(map[string]any); ok {
			if existingID, ok := getInt64(snap, "snapshot-id"); ok && existingID == snapshotID {
				return fmt.Errorf("snapshot-id %d already exists", snapshotID)
			}
		}
	}

	// Add snapshot to array
	p.metadata["snapshots"] = append(snapshots, u.Snapshot)

	// Update last-sequence-number if present in snapshot
	if seqNum, ok := getInt64(u.Snapshot, "sequence-number"); ok {
		currentSeq, _ := getInt64(p.metadata, "last-sequence-number")
		if seqNum > currentSeq {
			p.metadata["last-sequence-number"] = seqNum
		}
	}

	// Add entry to snapshot-log
	snapshotLog := ensureSlice(p.metadata, "snapshot-log")
	timestampMs, _ := getInt64(u.Snapshot, "timestamp-ms")
	p.metadata["snapshot-log"] = append(snapshotLog, map[string]any{
		"snapshot-id":  snapshotID,
		"timestamp-ms": timestampMs,
	})

	return nil
}

// applySetSnapshotRef creates or updates a snapshot reference (branch or tag).
// Branches are mutable pointers to snapshots; tags are immutable.
func (p *Processor) applySetSnapshotRef(u *SetSnapshotRef) error {
	if u.RefName == "" {
		return fmt.Errorf("ref-name is required")
	}

	if u.Type != refTypeBranch && u.Type != refTypeTag {
		return fmt.Errorf("type must be '%s' or '%s', got '%s'", refTypeBranch, refTypeTag, u.Type)
	}

	// Validate that the referenced snapshot exists (unless it's -1 for removal)
	if u.SnapshotID >= 0 {
		snapshots, _ := getSlice(p.metadata, "snapshots")
		found := false
		for _, s := range snapshots {
			if snap, ok := s.(map[string]any); ok {
				if id, ok := getInt64(snap, "snapshot-id"); ok && id == u.SnapshotID {
					found = true
					break
				}
			}
		}
		if !found {
			return fmt.Errorf("snapshot-id %d does not exist", u.SnapshotID)
		}
	}

	// Get or initialize refs map
	refs := ensureMap(p.metadata, "refs")

	// Build reference object
	ref := map[string]any{
		"type":        u.Type,
		"snapshot-id": u.SnapshotID,
	}

	// Add optional retention fields
	setIfNotNil(ref, "max-ref-age-ms", u.MaxRefAgeMs)
	setIfNotNil(ref, "max-snapshot-age-ms", u.MaxSnapshotAgeMs)
	setIfNotNil(ref, "min-snapshots-to-keep", u.MinSnapshotsToKeep)

	refs[u.RefName] = ref

	// If setting the "main" branch, also update current-snapshot-id
	if u.RefName == "main" && u.Type == refTypeBranch {
		p.metadata["current-snapshot-id"] = u.SnapshotID
	}

	return nil
}

// applyRemoveSnapshots removes snapshots by their IDs.
// This is used for snapshot expiration/cleanup.
func (p *Processor) applyRemoveSnapshots(u *RemoveSnapshots) error {
	if len(u.SnapshotIDs) == 0 {
		return nil
	}

	snapshots, ok := getSlice(p.metadata, "snapshots")
	if !ok || len(snapshots) == 0 {
		return nil
	}

	// Build set of IDs to remove for O(1) lookup
	toRemove := make(map[int64]bool, len(u.SnapshotIDs))
	for _, id := range u.SnapshotIDs {
		toRemove[id] = true
	}

	// Check that we're not removing a referenced snapshot
	refs, _ := getMap(p.metadata, "refs")
	for refName, refVal := range refs {
		if ref, ok := refVal.(map[string]any); ok {
			if refSnapshotID, ok := getInt64(ref, "snapshot-id"); ok {
				if toRemove[refSnapshotID] {
					return fmt.Errorf("cannot remove snapshot %d: referenced by '%s'", refSnapshotID, refName)
				}
			}
		}
	}

	// Check current-snapshot-id
	if currentID, ok := getInt64(p.metadata, "current-snapshot-id"); ok {
		if toRemove[currentID] {
			return fmt.Errorf("cannot remove current snapshot %d", currentID)
		}
	}

	// Filter snapshots
	filtered := filterSlice(snapshots, func(item any) bool {
		if snap, ok := item.(map[string]any); ok {
			if id, ok := getInt64(snap, "snapshot-id"); ok {
				return !toRemove[id]
			}
		}
		return true // Keep items we can't parse
	})

	p.metadata["snapshots"] = filtered
	return nil
}

// applyRemoveSnapshotRef removes a snapshot reference by name.
func (p *Processor) applyRemoveSnapshotRef(u *RemoveSnapshotRef) error {
	if u.RefName == "" {
		return fmt.Errorf("ref-name is required")
	}

	refs, ok := getMap(p.metadata, "refs")
	if !ok {
		return nil // No refs to remove from
	}

	// Check if removing "main" branch - need to clear current-snapshot-id
	if ref, ok := refs[u.RefName].(map[string]any); ok {
		if refType, _ := getString(ref, "type"); refType == "branch" && u.RefName == "main" {
			delete(p.metadata, "current-snapshot-id")
		}
	}

	delete(refs, u.RefName)
	return nil
}
