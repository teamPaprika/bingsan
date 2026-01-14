package updates

import "fmt"

// ============================================================================
// Sort Order Management
// ============================================================================

// applyAddSortOrder adds a new sort order to the table.
func (p *Processor) applyAddSortOrder(u *AddSortOrder) error {
	if u.SortOrder == nil {
		return fmt.Errorf("sort-order is required")
	}

	sortOrders := ensureSlice(p.metadata, "sort-orders")

	// Determine order-id for the new sort order
	var orderID int
	if existingID, ok := getInt(u.SortOrder, "order-id"); ok {
		orderID = existingID
	} else {
		// Auto-assign next order ID
		orderID = maxIntInSlice(sortOrders, "order-id") + 1
		u.SortOrder["order-id"] = orderID
	}

	// Check for duplicate order ID
	for _, s := range sortOrders {
		if order, ok := s.(map[string]any); ok {
			if existingID, ok := getInt(order, "order-id"); ok && existingID == orderID {
				return fmt.Errorf("order-id %d already exists", orderID)
			}
		}
	}

	p.metadata["sort-orders"] = append(sortOrders, u.SortOrder)
	return nil
}

// applySetDefaultSortOrder sets the default sort order.
// A sort-order-id of -1 means use the last added sort order.
func (p *Processor) applySetDefaultSortOrder(u *SetDefaultSortOrder) error {
	sortOrders, ok := getSlice(p.metadata, "sort-orders")
	if !ok || len(sortOrders) == 0 {
		return fmt.Errorf("no sort orders available")
	}

	targetID := u.SortOrderID

	// -1 means use the last added sort order
	if targetID == -1 {
		lastOrder := sortOrders[len(sortOrders)-1]
		if order, ok := lastOrder.(map[string]any); ok {
			if id, ok := getInt(order, "order-id"); ok {
				targetID = id
			}
		}
	}

	// Validate sort order exists
	found := false
	for _, s := range sortOrders {
		if order, ok := s.(map[string]any); ok {
			if id, ok := getInt(order, "order-id"); ok && id == targetID {
				found = true
				break
			}
		}
	}

	if !found {
		return fmt.Errorf("sort-order-id %d does not exist", targetID)
	}

	p.metadata["default-sort-order-id"] = targetID
	return nil
}
