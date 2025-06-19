// Package courierrepo provides data transfer objects and mapping functions for courier persistence.
// This package implements the repository pattern for the courier domain aggregate, handling
// the conversion between domain entities and database representations.
package courierrepo

import (
	"delivery/internal/core/domain/model/courier"
	"delivery/internal/core/domain/model/kernel"

	"github.com/google/uuid"
)

// CourierDTO represents the database structure for persisting courier aggregates.
// Maps courier domain entities to relational database tables with proper foreign key relationships.
type CourierDTO struct {
	ID            uuid.UUID         `gorm:"type:uuid;primaryKey"`
	Name          string            `gorm:"type:varchar(255);not null"`
	Speed         int               `gorm:"type:int;not null"`
	Location      LocationDTO       `gorm:"embedded;embeddedPrefix:location_"`
	StoragePlaces []StoragePlaceDTO `gorm:"foreignKey:CourierID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the database table name for courier entities.
// Overrides GORM's default naming convention to use "couriers" instead of "courier_dtos".
func (CourierDTO) TableName() string {
	return "couriers"
}

// LocationDTO represents the embedded location coordinates within the courier table.
// Stores the courier's current position on the delivery grid.
type LocationDTO struct {
	X kernel.Coordinate `gorm:"type:smallint"`
	Y kernel.Coordinate `gorm:"type:smallint"`
}

// StoragePlaceDTO represents the database structure for persisting storage place entities.
// Links to courier via foreign key and optionally references stored orders.
type StoragePlaceDTO struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"`
	CourierID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	Name        string     `gorm:"type:varchar(255);not null"`
	TotalVolume int        `gorm:"type:int;not null"`
	OrderID     *uuid.UUID `gorm:"type:uuid;index"`
}

// TableName specifies the database table name for storage place entities.
// Overrides GORM's default naming convention to use "storage_places" instead of "storage_place_dtos".
func (StoragePlaceDTO) TableName() string {
	return "storage_places"
}

// fromDomain converts a courier domain aggregate to its database representation.
// Maps all aggregate entities including storage places and their current state.
func fromDomain(courier *courier.Courier) CourierDTO {
	courierID := courier.ID().Bytes()
	storagePlaces := make([]StoragePlaceDTO, 0, len(courier.StoragePlaces()))

	for _, sp := range courier.StoragePlaces() {
		var orderID *uuid.UUID
		if sp.OrderID() != nil {
			raw := sp.OrderID().Bytes()
			orderID = &raw
		}

		storagePlaces = append(storagePlaces, StoragePlaceDTO{
			ID:          sp.ID().Bytes(),
			CourierID:   courierID,
			Name:        sp.Name(),
			TotalVolume: sp.TotalVolume(),
			OrderID:     orderID,
		})
	}

	return CourierDTO{
		ID:    courierID,
		Name:  courier.Name(),
		Speed: courier.Speed(),
		Location: LocationDTO{
			X: courier.Location().X(),
			Y: courier.Location().Y(),
		},
		StoragePlaces: storagePlaces,
	}
}

// toDomain converts a database DTO to a courier domain aggregate.
// Reconstructs the complete aggregate including all storage places using RestoreCourier.
func toDomain(dto CourierDTO) (*courier.Courier, error) {
	id, err := kernel.UUIDFromBytes(dto.ID[:])
	if err != nil {
		return nil, err
	}

	loc, err := kernel.NewLocation(dto.Location.X, dto.Location.Y)
	if err != nil {
		return nil, err
	}

	// Convert storage places DTOs to domain objects
	storagePlaces := make([]*courier.StoragePlace, 0, len(dto.StoragePlaces))
	for _, spDto := range dto.StoragePlaces {
		sp, spErr := storageplaceToDomain(spDto)
		if spErr != nil {
			return nil, spErr
		}
		storagePlaces = append(storagePlaces, sp)
	}

	return courier.RestoreCourier(id, dto.Name, dto.Speed, loc, storagePlaces)
}

// storageplaceToDomain converts a storage place DTO to domain entity.
// Uses RestoreStoragePlace to reconstruct the entity with its persisted state.
func storageplaceToDomain(dto StoragePlaceDTO) (*courier.StoragePlace, error) {
	id, err := kernel.UUIDFromBytes(dto.ID[:])
	if err != nil {
		return nil, err
	}

	var orderID *kernel.UUID
	if dto.OrderID != nil {
		oID, orderErr := kernel.UUIDFromBytes((*dto.OrderID)[:])
		if orderErr != nil {
			return nil, orderErr
		}
		orderID = &oID
	}

	return courier.RestoreStoragePlace(id, dto.Name, dto.TotalVolume, orderID)
}
