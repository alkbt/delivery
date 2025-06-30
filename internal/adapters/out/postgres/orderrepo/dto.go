// Package orderrepo provides data transfer objects and mapping functions for order persistence.
// This package implements the repository pattern for the order domain aggregate, handling
// the conversion between domain entities and database representations.
package orderrepo

import (
	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/core/domain/model/order"

	"github.com/google/uuid"
)

// OrderDTO represents the database structure for persisting order aggregates.
// Maps order domain entities to relational database tables with proper indexing
// for efficient querying by status and courier assignment.
type OrderDTO struct {
	ID        uuid.UUID   `gorm:"type:uuid;primaryKey"`
	CourierID *uuid.UUID  `gorm:"type:uuid;index"`
	Location  LocationDTO `gorm:"embedded;embeddedPrefix:location_"`
	Volume    int
	Status    int
}

// TableName specifies the database table name for order entities.
// Overrides GORM's default naming convention to use "orders".
func (OrderDTO) TableName() string {
	return "orders"
}

// LocationDTO represents the embedded delivery location coordinates within the order table.
// Stores the destination coordinates for order delivery.
type LocationDTO struct {
	X kernel.Coordinate `gorm:"type:smallint"`
	Y kernel.Coordinate `gorm:"type:smallint"`
}

// fromDomain converts an order domain aggregate to its database representation.
// Maps all order attributes including optional courier assignment.
func fromDomain(order *order.Order) OrderDTO {
	var courierID *uuid.UUID
	if id := order.Courier(); id != nil {
		raw := id.Bytes()
		courierID = &raw
	}

	return OrderDTO{
		ID:        order.ID().Bytes(),
		CourierID: courierID,
		Location: LocationDTO{
			X: order.Location().X(),
			Y: order.Location().Y(),
		},
		Volume: order.Volume(),
		Status: int(order.Status()),
	}
}

// toDomain converts a database DTO to an order domain aggregate.
// Reconstructs the complete aggregate including status and courier assignment using RestoreOrder.
func toDomain(dto OrderDTO) (*order.Order, error) {
	id, err := kernel.UUIDFromBytes(dto.ID[:])
	if err != nil {
		return nil, err
	}

	var courierID *kernel.UUID
	if dto.CourierID != nil {
		cID, courierErr := kernel.UUIDFromBytes((*dto.CourierID)[:])
		if courierErr != nil {
			return nil, courierErr
		}

		courierID = &cID
	}

	loc, err := kernel.NewLocation(dto.Location.X, dto.Location.Y)
	if err != nil {
		return nil, err
	}

	return order.RestoreOrder(id, loc, dto.Volume, order.Status(dto.Status), courierID)
}
