package queries

import (
	"errors"

	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/pkg/guard"
)

var (
	ErrGetUncompletedOrdersQueryIsNotConstructed = errors.New(
		"GetUncompletedOrdersQuery must be created via NewGetUncompletedOrdersQuery constructor",
	)
)

// GetUncompletedOrdersQuery retrieves all orders pending delivery.
// Returns orders in "created" or "assigned" status for monitoring and management.
//
// Example:
//
//	query := NewGetUncompletedOrdersQuery()
//	handler := NewGetUncompletedOrdersQueryHandler(db)
//
//	orders, err := handler.Handle(ctx, query)
//	if err != nil {
//	    return fmt.Errorf("failed to get pending orders: %w", err)
//	}
//
//	fmt.Printf("Found %d orders awaiting delivery\n", len(orders))
//	for _, order := range orders {
//	    fmt.Printf("Order %s at (%.2f, %.2f)\n",
//	        order.ID, order.Location.X(), order.Location.Y())
//	}
type GetUncompletedOrdersQuery struct {
	guard guard.ConstructorGuard
}

// NewGetUncompletedOrdersQuery creates a query to retrieve pending orders.
// This is a parameterless query that fetches all non-completed orders.
func NewGetUncompletedOrdersQuery() GetUncompletedOrdersQuery {
	return GetUncompletedOrdersQuery{guard: guard.NewConstructorGuard()}
}

// Validate ensures the query was created through the constructor.
// Returns ErrGetUncompletedOrdersQueryIsNotConstructed if validation fails.
func (q GetUncompletedOrdersQuery) Validate() error {
	return q.guard.Validate(ErrGetUncompletedOrdersQueryIsNotConstructed)
}

// GetUncompletedOrdersQueryResponse represents pending order information.
// Contains essential data for delivery tracking and courier assignment.
//
// Example:
//
//	response := GetUncompletedOrdersQueryResponse{
//	    ID:       kernel.MustNewUUID("123e4567-e89b-12d3-a456-426614174000"),
//	    Location: kernel.NewLocation(40.7128, -74.0060), // New York
//	}
type GetUncompletedOrdersQueryResponse struct {
	ID       kernel.UUID
	Location kernel.Location
}
