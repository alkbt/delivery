// Package ports defines repository interfaces for the courier domain.
// These interfaces establish contracts between the domain layer and infrastructure,
// enabling dependency inversion and testability.
package ports

import (
	"context"

	"delivery/internal/core/domain/model/courier"
	"delivery/internal/core/domain/model/kernel"
)

// CourierRepository defines the persistence contract for courier aggregates.
// Provides methods for storing, retrieving, and querying courier entities
// with their complete state including storage places.
type CourierRepository interface {
	// Add persists a new courier aggregate to storage.
	// The courier must be valid and not already exist in the repository.
	Add(ctx context.Context, courier *courier.Courier) error

	// Update persists changes to an existing courier aggregate.
	// The courier must exist in the repository and be valid.
	Update(ctx context.Context, courier *courier.Courier) error

	// Get retrieves a courier aggregate by its unique identifier.
	// Returns the complete courier with all storage places and their current state.
	Get(ctx context.Context, id kernel.UUID) (*courier.Courier, error)

	// GetAllFree retrieves all couriers that are not currently assigned to active orders.
	// A courier is considered free if they are not assigned to any order in Assigned status.
	// Couriers with Created orders (not yet assigned) or Completed orders (finished deliveries)
	// are considered available for new assignments.
	//
	// Business Rules:
	//   - Couriers without any orders: Available
	//   - Couriers with Created orders: Available (orders not assigned yet)
	//   - Couriers with Assigned orders: Unavailable (actively working)
	//   - Couriers with Completed orders: Available (work finished)
	//
	// Example:
	//   freeCouriers, err := repo.GetAllFree(ctx)
	//   if err != nil {
	//       return fmt.Errorf("failed to get available couriers: %w", err)
	//   }
	//   for _, courier := range freeCouriers {
	//       fmt.Printf("Available: %s\n", courier.Name())
	//   }
	GetAllFree(ctx context.Context) ([]*courier.Courier, error)
}
