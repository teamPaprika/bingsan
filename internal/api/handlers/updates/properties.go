package updates

import "fmt"

// ============================================================================
// Core Table Operations
// ============================================================================

// applyAssignUUID assigns a UUID to the table.
// This is typically only used during table creation.
func (p *Processor) applyAssignUUID(u *AssignUUID) error {
	if u.UUID == "" {
		return fmt.Errorf("uuid is required")
	}

	// Only allow if not already set or if setting to same value
	existingUUID, exists := getString(p.metadata, "table-uuid")
	if exists && existingUUID != "" && existingUUID != u.UUID {
		return fmt.Errorf("cannot change table-uuid once set")
	}

	p.metadata["table-uuid"] = u.UUID
	return nil
}

// applyUpgradeFormatVersion upgrades the table format version.
func (p *Processor) applyUpgradeFormatVersion(u *UpgradeFormatVersion) error {
	if u.FormatVersion < 1 || u.FormatVersion > 3 {
		return fmt.Errorf("format-version must be 1, 2, or 3")
	}

	currentVersion, _ := getInt(p.metadata, "format-version")
	if u.FormatVersion < currentVersion {
		return fmt.Errorf("cannot downgrade format-version from %d to %d", currentVersion, u.FormatVersion)
	}

	p.metadata["format-version"] = u.FormatVersion
	return nil
}

// applySetLocation changes the table's base location.
func (p *Processor) applySetLocation(u *SetLocation) error {
	if u.Location == "" {
		return fmt.Errorf("location is required")
	}

	p.metadata["location"] = u.Location
	return nil
}

// applySetProperties sets or updates table properties.
func (p *Processor) applySetProperties(u *SetProperties) error {
	if len(u.Updates) == 0 {
		return nil
	}

	props := ensureMap(p.metadata, "properties")

	for key, value := range u.Updates {
		props[key] = value
	}

	return nil
}

// applyRemoveProperties removes table properties by key.
func (p *Processor) applyRemoveProperties(u *RemoveProperties) error {
	if len(u.Removals) == 0 {
		return nil
	}

	props, ok := getMap(p.metadata, "properties")
	if !ok {
		return nil // Nothing to remove
	}

	for _, key := range u.Removals {
		delete(props, key)
	}

	return nil
}

// ============================================================================
// Encryption Key Management
// ============================================================================

// applyAddEncryptionKey adds an encryption key to the table.
func (p *Processor) applyAddEncryptionKey(u *AddEncryptionKey) error {
	if u.EncryptionKey == nil {
		return fmt.Errorf("encryption-key is required")
	}

	keyID, ok := getString(u.EncryptionKey, "key-id")
	if !ok || keyID == "" {
		return fmt.Errorf("key-id is required in encryption-key")
	}

	keys := ensureSlice(p.metadata, "encryption-keys")

	// Check for duplicate key ID
	for _, k := range keys {
		if key, ok := k.(map[string]any); ok {
			if existingID, ok := getString(key, "key-id"); ok && existingID == keyID {
				return fmt.Errorf("encryption key-id %s already exists", keyID)
			}
		}
	}

	p.metadata["encryption-keys"] = append(keys, u.EncryptionKey)
	return nil
}

// applyRemoveEncryptionKey removes an encryption key by ID.
func (p *Processor) applyRemoveEncryptionKey(u *RemoveEncryptionKey) error {
	if u.KeyID == "" {
		return fmt.Errorf("key-id is required")
	}

	keys, ok := getSlice(p.metadata, "encryption-keys")
	if !ok || len(keys) == 0 {
		return nil
	}

	filtered := filterSlice(keys, func(item any) bool {
		if key, ok := item.(map[string]any); ok {
			if id, ok := getString(key, "key-id"); ok {
				return id != u.KeyID
			}
		}
		return true
	})

	p.metadata["encryption-keys"] = filtered
	return nil
}
