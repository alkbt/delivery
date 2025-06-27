package courierrepo

import (
	"context"
	"errors"

	"delivery/internal/core/domain/model/courier"
	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/core/domain/model/order"
	"delivery/internal/pkg/errs"

	"gorm.io/gorm"
)

// GormCourierRepository implements CourierRepository using GORM.
type GormCourierRepository struct {
	db      *gorm.DB
	tracker aggregateTracker
}

// aggregateTracker defines the interface for tracking aggregates.
type aggregateTracker interface {
	TrackAggregate(id kernel.UUID, aggregate any)
}

// NewGormCourierRepository creates a new GORM courier repository.
func NewGormCourierRepository(db *gorm.DB, tracker aggregateTracker) *GormCourierRepository {
	return &GormCourierRepository{
		db:      db,
		tracker: tracker,
	}
}

// Add saves a new courier to the database.
func (r *GormCourierRepository) Add(ctx context.Context, aggregate *courier.Courier) error {
	if err := aggregate.Validate(); err != nil {
		return err
	}

	dto := fromDomain(aggregate)
	if err := r.db.WithContext(ctx).Create(&dto).Error; err != nil {
		return err
	}

	r.tracker.TrackAggregate(aggregate.ID(), aggregate)
	return nil
}

// Update saves an existing courier to the database.
func (r *GormCourierRepository) Update(ctx context.Context, aggregate *courier.Courier) error {
	if err := aggregate.Validate(); err != nil {
		return err
	}

	dto := fromDomain(aggregate)

	// Use Session with FullSaveAssociations to properly update nested associations
	result := r.db.WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Save(&dto)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	r.tracker.TrackAggregate(aggregate.ID(), aggregate)
	return nil
}

// Get retrieves a courier by ID.
func (r *GormCourierRepository) Get(ctx context.Context, id kernel.UUID) (*courier.Courier, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}

	var dto CourierDTO
	if err := r.db.WithContext(ctx).Preload("StoragePlaces").First(&dto, "id = ?", id.Bytes()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NewObjectNotFoundError("courier", id.String())
		}
		return nil, err
	}

	return toDomain(dto)
}

// GetAllFree retrieves all couriers that are not currently assigned to active orders.
// A courier is considered free if they are not assigned to any order in Assigned status.
// Orders in Created status don't have couriers assigned yet, and orders in Completed
// status have finished, so their couriers are available again.
//
// Example:
//
//	freeCouriers, err := repo.GetAllFree(ctx)
//	if err != nil {
//		return fmt.Errorf("failed to get free couriers: %w", err)
//	}
//	for _, courier := range freeCouriers {
//		fmt.Printf("Available courier: %s\n", courier.Name())
//	}
func (r *GormCourierRepository) GetAllFree(ctx context.Context) ([]*courier.Courier, error) {
	var dtos []CourierDTO
	// Join with orders table to find couriers not assigned to any orders in Assigned status
	if err := r.db.WithContext(ctx).
		Preload("StoragePlaces").
		Table("couriers").
		Select("couriers.*").
		Joins("LEFT JOIN orders ON couriers.id = orders.courier_id AND orders.status = ?", int(order.Assigned)).
		Where("orders.courier_id IS NULL").
		Find(&dtos).Error; err != nil {
		return nil, err
	}

	couriers := make([]*courier.Courier, 0, len(dtos))
	for _, dto := range dtos {
		c, err := toDomain(dto)
		if err != nil {
			return nil, err
		}
		couriers = append(couriers, c)
	}

	return couriers, nil
}
