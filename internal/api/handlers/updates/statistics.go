package updates

import "fmt"

// ============================================================================
// Statistics Management
// ============================================================================

// applySetStatistics sets or updates table statistics for a snapshot.
func (p *Processor) applySetStatistics(u *SetStatistics) error {
	if u.Statistics == nil {
		return fmt.Errorf("statistics is required")
	}

	// Get snapshot-id from statistics object or deprecated field
	var snapshotID int64
	if sid, ok := getInt64(u.Statistics, "snapshot-id"); ok {
		snapshotID = sid
	} else if u.SnapshotID != nil {
		snapshotID = *u.SnapshotID
		// Set it in the statistics object for consistency
		u.Statistics["snapshot-id"] = snapshotID
	} else {
		return fmt.Errorf("snapshot-id is required in statistics")
	}

	statistics := ensureSlice(p.metadata, "statistics")

	// Remove existing statistics for this snapshot (replace semantics)
	filtered := filterSlice(statistics, func(item any) bool {
		if stat, ok := item.(map[string]any); ok {
			if existingID, ok := getInt64(stat, "snapshot-id"); ok {
				return existingID != snapshotID
			}
		}
		return true
	})

	p.metadata["statistics"] = append(filtered, u.Statistics)
	return nil
}

// applyRemoveStatistics removes statistics for a snapshot.
func (p *Processor) applyRemoveStatistics(u *RemoveStatistics) error {
	statistics, ok := getSlice(p.metadata, "statistics")
	if !ok || len(statistics) == 0 {
		return nil
	}

	filtered := filterSlice(statistics, func(item any) bool {
		if stat, ok := item.(map[string]any); ok {
			if existingID, ok := getInt64(stat, "snapshot-id"); ok {
				return existingID != u.SnapshotID
			}
		}
		return true
	})

	p.metadata["statistics"] = filtered
	return nil
}

// applySetPartitionStatistics sets partition-level statistics.
func (p *Processor) applySetPartitionStatistics(u *SetPartitionStatistics) error {
	if u.PartitionStatistics == nil {
		return fmt.Errorf("partition-statistics is required")
	}

	snapshotID, ok := getInt64(u.PartitionStatistics, "snapshot-id")
	if !ok {
		return fmt.Errorf("snapshot-id is required in partition-statistics")
	}

	partitionStats := ensureSlice(p.metadata, "partition-statistics")

	// Remove existing partition statistics for this snapshot (replace semantics)
	filtered := filterSlice(partitionStats, func(item any) bool {
		if stat, ok := item.(map[string]any); ok {
			if existingID, ok := getInt64(stat, "snapshot-id"); ok {
				return existingID != snapshotID
			}
		}
		return true
	})

	p.metadata["partition-statistics"] = append(filtered, u.PartitionStatistics)
	return nil
}

// applyRemovePartitionStatistics removes partition statistics for a snapshot.
func (p *Processor) applyRemovePartitionStatistics(u *RemovePartitionStatistics) error {
	partitionStats, ok := getSlice(p.metadata, "partition-statistics")
	if !ok || len(partitionStats) == 0 {
		return nil
	}

	filtered := filterSlice(partitionStats, func(item any) bool {
		if stat, ok := item.(map[string]any); ok {
			if existingID, ok := getInt64(stat, "snapshot-id"); ok {
				return existingID != u.SnapshotID
			}
		}
		return true
	})

	p.metadata["partition-statistics"] = filtered
	return nil
}
