package queries

import (
	"context"

	"delivery/internal/core/domain/model/kernel"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetAllCouriersQueryHandler retrieves all courier information from the database.
// Uses direct SQL queries for optimal read performance in the CQRS pattern.
//
// Example:
//
//	handler := NewGetAllCouriersQueryHandler(db)
//	query := NewGetAllCouriersQuery()
//
//	couriers, err := handler.Handle(ctx, query)
//	if err != nil {
//	    log.Printf("Failed to get couriers: %v", err)
//	    return err
//	}
//
//	fmt.Printf("Found %d couriers\n", len(couriers))
type GetAllCouriersQueryHandler struct {
	db *gorm.DB
}

// NewGetAllCouriersQueryHandler creates a handler for courier retrieval queries.
// Requires a GORM database connection for query execution.
func NewGetAllCouriersQueryHandler(db *gorm.DB) GetAllCouriersQueryHandler {
	return GetAllCouriersQueryHandler{db: db}
}

// Handle executes the query to retrieve all couriers.
// Returns a slice of courier read models sorted by name.
// Converts database types to domain types for consistency.
func (h GetAllCouriersQueryHandler) Handle(
	ctx context.Context,
	query GetAllCouriersQuery,
) ([]GetAllCouriersQueryResponse, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}

	couriers := make([]GetAllCouriersQueryResponse, 0)

	rows, err := h.db.WithContext(ctx).Raw(`
		SELECT 
			id, 
			name, 
			location_x, 
			location_y 
		FROM couriers
		ORDER BY name
	`).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var courier GetAllCouriersQueryResponse
		var locationX, locationY int8
		var id uuid.UUID

		err = rows.Scan(
			&id,
			&courier.Name,
			&locationX,
			&locationY,
		)
		if err != nil {
			return nil, err
		}

		courierID, idErr := kernel.UUIDFromBytes(id[:])
		if idErr != nil {
			return nil, idErr
		}
		courier.ID = courierID

		location, locErr := kernel.NewLocation(
			kernel.Coordinate(locationX),
			kernel.Coordinate(locationY),
		)
		if locErr != nil {
			return nil, locErr
		}
		courier.Location = location
		couriers = append(couriers, courier)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return couriers, nil
}
