package updates

import "fmt"

// ============================================================================
// Schema Management
// ============================================================================

// applyAddSchema adds a new schema version to the table.
func (p *Processor) applyAddSchema(u *AddSchema) error {
	if u.Schema == nil {
		return fmt.Errorf("schema is required")
	}

	schemas := ensureSlice(p.metadata, "schemas")

	// Determine schema-id for the new schema
	var schemaID int
	if existingID, ok := getInt(u.Schema, "schema-id"); ok {
		schemaID = existingID
	} else {
		// Auto-assign next schema ID
		schemaID = maxIntInSlice(schemas, "schema-id") + 1
		u.Schema["schema-id"] = schemaID
	}

	// Check for duplicate schema ID
	for _, s := range schemas {
		if schema, ok := s.(map[string]any); ok {
			if existingID, ok := getInt(schema, "schema-id"); ok && existingID == schemaID {
				return fmt.Errorf("schema-id %d already exists", schemaID)
			}
		}
	}

	// Update last-column-id from schema fields
	if fields, ok := u.Schema["fields"].([]any); ok {
		maxFieldID := 0
		for _, f := range fields {
			if field, ok := f.(map[string]any); ok {
				if id, ok := getInt(field, "id"); ok && id > maxFieldID {
					maxFieldID = id
				}
			}
		}

		// Update table's last-column-id if this schema has higher field IDs
		currentLastColumnID, _ := getInt(p.metadata, "last-column-id")
		if maxFieldID > currentLastColumnID {
			p.metadata["last-column-id"] = maxFieldID
		}
	}

	// Handle deprecated last-column-id in update
	if u.LastColumnID != nil {
		currentLastColumnID, _ := getInt(p.metadata, "last-column-id")
		if *u.LastColumnID > currentLastColumnID {
			p.metadata["last-column-id"] = *u.LastColumnID
		}
	}

	p.metadata["schemas"] = append(schemas, u.Schema)
	return nil
}

// applySetCurrentSchema sets the current schema by ID.
// A schema-id of -1 means use the last added schema.
func (p *Processor) applySetCurrentSchema(u *SetCurrentSchema) error {
	schemas, ok := getSlice(p.metadata, "schemas")
	if !ok || len(schemas) == 0 {
		return fmt.Errorf("no schemas available")
	}

	targetID := u.SchemaID

	// -1 means use the last added schema
	if targetID == -1 {
		lastSchema := schemas[len(schemas)-1]
		if schema, ok := lastSchema.(map[string]any); ok {
			if id, ok := getInt(schema, "schema-id"); ok {
				targetID = id
			}
		}
	}

	// Validate schema exists
	found := false
	for _, s := range schemas {
		if schema, ok := s.(map[string]any); ok {
			if id, ok := getInt(schema, "schema-id"); ok && id == targetID {
				found = true
				break
			}
		}
	}

	if !found {
		return fmt.Errorf("schema-id %d does not exist", targetID)
	}

	p.metadata["current-schema-id"] = targetID
	return nil
}

// applyRemoveSchemas removes schema versions by ID.
func (p *Processor) applyRemoveSchemas(u *RemoveSchemas) error {
	if len(u.SchemaIDs) == 0 {
		return nil
	}

	schemas, ok := getSlice(p.metadata, "schemas")
	if !ok || len(schemas) == 0 {
		return nil
	}

	// Build set of IDs to remove
	toRemove := make(map[int]bool, len(u.SchemaIDs))
	for _, id := range u.SchemaIDs {
		toRemove[id] = true
	}

	// Check we're not removing the current schema
	currentSchemaID, _ := getInt(p.metadata, "current-schema-id")
	if toRemove[currentSchemaID] {
		return fmt.Errorf("cannot remove current schema %d", currentSchemaID)
	}

	// Filter schemas
	filtered := filterSlice(schemas, func(item any) bool {
		if schema, ok := item.(map[string]any); ok {
			if id, ok := getInt(schema, "schema-id"); ok {
				return !toRemove[id]
			}
		}
		return true
	})

	if len(filtered) == 0 {
		return fmt.Errorf("cannot remove all schemas")
	}

	p.metadata["schemas"] = filtered
	return nil
}
