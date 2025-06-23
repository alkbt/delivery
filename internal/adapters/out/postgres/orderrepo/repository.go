package orderrepo

import (
	"context"
	"errors"

	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/core/domain/model/order"
	"delivery/internal/pkg/errs"

	"gorm.io/gorm"
)

// GormOrderRepository implements OrderRepository using GORM.
type GormOrderRepository struct {
	db      *gorm.DB
	tracker aggregateTracker
}

// aggregateTracker defines the interface for tracking aggregates.
type aggregateTracker interface {
	TrackAggregate(id kernel.UUID, aggregate any)
}

// NewGormOrderRepository creates a new GORM order repository.
func NewGormOrderRepository(db *gorm.DB, tracker aggregateTracker) *GormOrderRepository {
	return &GormOrderRepository{
		db:      db,
		tracker: tracker,
	}
}

// Add saves a new order to the database.
func (r *GormOrderRepository) Add(ctx context.Context, aggregate *order.Order) error {
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

// Update saves an existing order to the database.
func (r *GormOrderRepository) Update(ctx context.Context, aggregate *order.Order) error {
	if err := aggregate.Validate(); err != nil {
		return err
	}

	dto := fromDomain(aggregate)
	result := r.db.WithContext(ctx).Model(&OrderDTO{}).Where("id = ?", dto.ID).Updates(&dto)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	r.tracker.TrackAggregate(aggregate.ID(), aggregate)
	return nil
}

// Get retrieves an order by ID.
func (r *GormOrderRepository) Get(ctx context.Context, id kernel.UUID) (*order.Order, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}

	var dto OrderDTO
	if err := r.db.WithContext(ctx).First(&dto, "id = ?", id.Bytes()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NewObjectNotFoundError("order", id.String())
		}
		return nil, err
	}

	return toDomain(dto)
}

// GetFirstInCreatedStatus retrieves the first order with Created status.
func (r *GormOrderRepository) GetFirstInCreatedStatus(ctx context.Context) (*order.Order, error) {
	var dto OrderDTO
	if err := r.db.WithContext(ctx).First(&dto, "status = ?", order.Created).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NewObjectNotFoundError("order", "first in created status")
		}
		return nil, err
	}

	return toDomain(dto)
}

// GetAllInAssignedStatus retrieves all orders with Assigned status.
func (r *GormOrderRepository) GetAllInAssignedStatus(ctx context.Context) ([]*order.Order, error) {
	var dtos []OrderDTO
	if err := r.db.WithContext(ctx).Find(&dtos, "status = ?", order.Assigned).Error; err != nil {
		return nil, err
	}

	orders := make([]*order.Order, 0, len(dtos))
	for _, dto := range dtos {
		o, err := toDomain(dto)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}

	return orders, nil
}
