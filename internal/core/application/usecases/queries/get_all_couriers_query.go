// Package queries contains read operations for retrieving system state.
// Implements the Query pattern for read operations in the CQRS architecture.
// Queries return optimized read models for specific use cases.
package queries

import (
	"errors"

	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/pkg/guard"
)

var (
	ErrGetAllCouriersQueryIsNotConstructed = errors.New(
		"GetAllCouriersQuery must be created via NewGetAllCouriersQuery constructor",
	)
)

// GetAllCouriersQuery retrieves information about all couriers in the system.
// Returns courier identities and current locations for monitoring and dispatching.
//
// Example:
//
//	query := NewGetAllCouriersQuery()
//	handler := NewGetAllCouriersQueryHandler(db)
//
//	couriers, err := handler.Handle(ctx, query)
//	if err != nil {
//	    return fmt.Errorf("failed to retrieve couriers: %w", err)
//	}
//
//	for _, courier := range couriers {
//	    fmt.Printf("Courier %s at location (%.2f, %.2f)\n",
//	        courier.Name, courier.Location.X(), courier.Location.Y())
//	}
type GetAllCouriersQuery struct {
	guard guard.ConstructorGuard
}

// NewGetAllCouriersQuery creates a query to retrieve all couriers.
// This is a parameterless query that fetches the complete courier list.
func NewGetAllCouriersQuery() GetAllCouriersQuery {
	return GetAllCouriersQuery{guard: guard.NewConstructorGuard()}
}

// Validate ensures the query was created through the constructor.
// Returns ErrGetAllCouriersQueryIsNotConstructed if validation fails.
func (q GetAllCouriersQuery) Validate() error {
	return q.guard.Validate(ErrGetAllCouriersQueryIsNotConstructed)
}

// GetAllCouriersQueryResponse represents courier information in the read model.
// Contains essential courier data for display and decision-making.
//
// Example:
//
//	response := GetAllCouriersQueryResponse{
//	    ID:       kernel.MustNewUUID("550e8400-e29b-41d4-a716-446655440000"),
//	    Name:     "Express Courier",
//	    Location: kernel.NewLocation(55.7558, 37.6173),
//	}
type GetAllCouriersQueryResponse struct {
	ID       kernel.UUID
	Name     string
	Location kernel.Location
}
