package ports

import (
	"context"

	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/core/domain/model/order"
)

// OrderRepository defines the persistence contract for order aggregates.
// Provides methods for storing, retrieving, and querying order entities
// based on their status and assignment state.
type OrderRepository interface {
	// Add persists a new order aggregate to storage.
	// The order must be valid and not already exist in the repository.
	Add(ctx context.Context, aggregate *order.Order) error

	// Update persists changes to an existing order aggregate.
	// The order must exist in the repository and be valid.
	Update(ctx context.Context, aggregate *order.Order) error

	// Get retrieves an order aggregate by its unique identifier.
	// Returns the complete order with its current status and assignment.
	Get(ctx context.Context, id kernel.UUID) (*order.Order, error)

	// GetFirstInCreatedStatus retrieves the first order in Created status.
	// Used for order assignment workflows to find pending orders.
	GetFirstInCreatedStatus(ctx context.Context) (*order.Order, error)

	// GetAllInAssignedStatus retrieves all orders currently assigned to couriers.
	// Returns orders that are in progress but not yet completed.
	GetAllInAssignedStatus(ctx context.Context) ([]*order.Order, error)
}
