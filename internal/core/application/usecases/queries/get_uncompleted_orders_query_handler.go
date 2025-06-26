package queries

import (
	"context"

	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/core/domain/model/order"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetUncompletedOrdersQueryHandler retrieves orders pending delivery from the database.
// Filters out completed orders to provide active delivery workload visibility.
//
// Example:
//
//	handler := NewGetUncompletedOrdersQueryHandler(db)
//	query := NewGetUncompletedOrdersQuery()
//
//	pendingOrders, err := handler.Handle(ctx, query)
//	if err != nil {
//	    log.Printf("Failed to get pending orders: %v", err)
//	    return err
//	}
//
//	if len(pendingOrders) > 0 {
//	    fmt.Printf("%d orders awaiting delivery\n", len(pendingOrders))
//	}
type GetUncompletedOrdersQueryHandler struct {
	db *gorm.DB
}

// NewGetUncompletedOrdersQueryHandler creates a handler for pending order queries.
// Requires a GORM database connection for query execution.
func NewGetUncompletedOrdersQueryHandler(db *gorm.DB) GetUncompletedOrdersQueryHandler {
	return GetUncompletedOrdersQueryHandler{db: db}
}

// Handle executes the query to retrieve all uncompleted orders.
// Returns orders in "created" or "assigned" status, excluding completed deliveries.
// Results are sorted by order ID for consistent output.
func (h GetUncompletedOrdersQueryHandler) Handle(
	ctx context.Context,
	query GetUncompletedOrdersQuery,
) ([]GetUncompletedOrdersQueryResponse, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}

	orders := make([]GetUncompletedOrdersQueryResponse, 0)

	rows, err := h.db.WithContext(ctx).Raw(`
		SELECT 
			id, 
			location_x, 
			location_y 
		FROM orders
		WHERE status != ?
		ORDER BY id
	`, order.Completed).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var orderResp GetUncompletedOrdersQueryResponse
		var locationX, locationY int8
		var id uuid.UUID

		err = rows.Scan(
			&id,
			&locationX,
			&locationY,
		)
		if err != nil {
			return nil, err
		}

		orderID, idErr := kernel.UUIDFromBytes(id[:])
		if idErr != nil {
			return nil, idErr
		}
		orderResp.ID = orderID

		location, locErr := kernel.NewLocation(
			kernel.Coordinate(locationX),
			kernel.Coordinate(locationY),
		)
		if locErr != nil {
			return nil, locErr
		}
		orderResp.Location = location
		orders = append(orders, orderResp)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}
