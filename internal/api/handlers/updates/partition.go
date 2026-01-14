package updates

import "fmt"

// ============================================================================
// Partition Spec Management
// ============================================================================

// applyAddSpec adds a new partition specification.
func (p *Processor) applyAddSpec(u *AddSpec) error {
	if u.Spec == nil {
		return fmt.Errorf("spec is required")
	}

	specs := ensureSlice(p.metadata, "partition-specs")

	// Determine spec-id for the new spec
	var specID int
	if existingID, ok := getInt(u.Spec, "spec-id"); ok {
		specID = existingID
	} else {
		// Auto-assign next spec ID
		specID = maxIntInSlice(specs, "spec-id") + 1
		u.Spec["spec-id"] = specID
	}

	// Check for duplicate spec ID
	for _, s := range specs {
		if spec, ok := s.(map[string]any); ok {
			if existingID, ok := getInt(spec, "spec-id"); ok && existingID == specID {
				return fmt.Errorf("spec-id %d already exists", specID)
			}
		}
	}

	// Update last-partition-id from spec fields
	if fields, ok := u.Spec["fields"].([]any); ok {
		maxFieldID := 0
		for _, f := range fields {
			if field, ok := f.(map[string]any); ok {
				if id, ok := getInt(field, "field-id"); ok && id > maxFieldID {
					maxFieldID = id
				}
			}
		}

		currentLastPartitionID, _ := getInt(p.metadata, "last-partition-id")
		if maxFieldID > currentLastPartitionID {
			p.metadata["last-partition-id"] = maxFieldID
		}
	}

	p.metadata["partition-specs"] = append(specs, u.Spec)
	return nil
}

// applySetDefaultSpec sets the default partition spec.
// A spec-id of -1 means use the last added spec.
func (p *Processor) applySetDefaultSpec(u *SetDefaultSpec) error {
	specs, ok := getSlice(p.metadata, "partition-specs")
	if !ok || len(specs) == 0 {
		return fmt.Errorf("no partition specs available")
	}

	targetID := u.SpecID

	// -1 means use the last added spec
	if targetID == -1 {
		lastSpec := specs[len(specs)-1]
		if spec, ok := lastSpec.(map[string]any); ok {
			if id, ok := getInt(spec, "spec-id"); ok {
				targetID = id
			}
		}
	}

	// Validate spec exists
	found := false
	for _, s := range specs {
		if spec, ok := s.(map[string]any); ok {
			if id, ok := getInt(spec, "spec-id"); ok && id == targetID {
				found = true
				break
			}
		}
	}

	if !found {
		return fmt.Errorf("spec-id %d does not exist", targetID)
	}

	p.metadata["default-spec-id"] = targetID
	return nil
}

// applyRemovePartitionSpecs removes partition specs by ID.
func (p *Processor) applyRemovePartitionSpecs(u *RemovePartitionSpecs) error {
	if len(u.SpecIDs) == 0 {
		return nil
	}

	specs, ok := getSlice(p.metadata, "partition-specs")
	if !ok || len(specs) == 0 {
		return nil
	}

	// Build set of IDs to remove
	toRemove := make(map[int]bool, len(u.SpecIDs))
	for _, id := range u.SpecIDs {
		toRemove[id] = true
	}

	// Check we're not removing the default spec
	defaultSpecID, _ := getInt(p.metadata, "default-spec-id")
	if toRemove[defaultSpecID] {
		return fmt.Errorf("cannot remove default partition spec %d", defaultSpecID)
	}

	// Filter specs
	filtered := filterSlice(specs, func(item any) bool {
		if spec, ok := item.(map[string]any); ok {
			if id, ok := getInt(spec, "spec-id"); ok {
				return !toRemove[id]
			}
		}
		return true
	})

	if len(filtered) == 0 {
		return fmt.Errorf("cannot remove all partition specs")
	}

	p.metadata["partition-specs"] = filtered
	return nil
}
