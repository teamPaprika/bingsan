package updates

import (
	"encoding/json"
	"fmt"
)

// deepCopyMap creates a deep copy of a map using JSON round-trip.
// This ensures nested structures are properly copied.
func deepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}

	data, err := json.Marshal(m)
	if err != nil {
		// Fallback to shallow copy on marshal error (shouldn't happen)
		result := make(map[string]any, len(m))
		for k, v := range m {
			result[k] = v
		}
		return result
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}

// getInt64 safely extracts an int64 from a map value.
// Handles both float64 (from JSON) and native integer types.
func getInt64(m map[string]any, key string) (int64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	}
	return 0, false
}

// getInt safely extracts an int from a map value.
func getInt(m map[string]any, key string) (int, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int64:
		return int(n), true
	case int:
		return n, true
	case json.Number:
		i, err := n.Int64()
		return int(i), err == nil
	}
	return 0, false
}

// getString safely extracts a string from a map value.
func getString(m map[string]any, key string) (string, bool) {
	v, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// getSlice safely extracts a slice from a map value.
func getSlice(m map[string]any, key string) ([]any, bool) {
	v, ok := m[key]
	if !ok {
		return nil, false
	}
	s, ok := v.([]any)
	return s, ok
}

// getMap safely extracts a nested map from a map value.
func getMap(m map[string]any, key string) (map[string]any, bool) {
	v, ok := m[key]
	if !ok {
		return nil, false
	}
	nested, ok := v.(map[string]any)
	return nested, ok
}

// setIfNotNil sets a map key only if the value is not nil.
func setIfNotNil[T any](m map[string]any, key string, value *T) {
	if value != nil {
		m[key] = *value
	}
}

// ensureSlice ensures a key exists as a slice in the map.
// If the key doesn't exist or isn't a slice, initializes it as empty.
func ensureSlice(m map[string]any, key string) []any {
	if s, ok := m[key].([]any); ok {
		return s
	}
	s := []any{}
	m[key] = s
	return s
}

// ensureMap ensures a key exists as a map in the map.
// If the key doesn't exist or isn't a map, initializes it as empty.
func ensureMap(m map[string]any, key string) map[string]any {
	if nested, ok := m[key].(map[string]any); ok {
		return nested
	}
	nested := map[string]any{}
	m[key] = nested
	return nested
}

// filterSlice returns a new slice with elements matching the predicate.
func filterSlice(slice []any, predicate func(any) bool) []any {
	result := make([]any, 0, len(slice))
	for _, item := range slice {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// maxIntInSlice finds the maximum int value for a key in a slice of maps.
func maxIntInSlice(slice []any, key string) int {
	var maxVal int
	for _, item := range slice {
		if m, ok := item.(map[string]any); ok {
			if v, ok := getInt(m, key); ok && v > maxVal {
				maxVal = v
			}
		}
	}
	return maxVal
}

// removeItemsByID is a generic helper for removing items from a slice by their ID field.
// It returns the filtered slice and an error if validation fails.
// Parameters:
//   - items: the slice of map[string]any items
//   - idsToRemove: slice of IDs to remove
//   - idKey: the key name for the ID field (e.g., "schema-id", "spec-id")
//   - protectedID: the ID that cannot be removed (e.g., current schema, default spec)
//   - itemType: human-readable name for error messages (e.g., "schema", "partition spec")
func removeItemsByID(items []any, idsToRemove []int, idKey string, protectedID int, itemType string) ([]any, error) {
	if len(idsToRemove) == 0 || len(items) == 0 {
		return items, nil
	}

	// Build set of IDs to remove
	toRemove := make(map[int]bool, len(idsToRemove))
	for _, id := range idsToRemove {
		toRemove[id] = true
	}

	// Check we're not removing the protected item
	if toRemove[protectedID] {
		return nil, fmt.Errorf("cannot remove %s %d", itemType, protectedID)
	}

	// Filter items
	filtered := filterSlice(items, func(item any) bool {
		if m, ok := item.(map[string]any); ok {
			if id, ok := getInt(m, idKey); ok {
				return !toRemove[id]
			}
		}
		return true
	})

	if len(filtered) == 0 {
		return nil, fmt.Errorf("cannot remove all %ss", itemType)
	}

	return filtered, nil
}
