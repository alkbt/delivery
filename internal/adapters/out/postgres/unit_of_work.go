// Package postgres provides GORM-based implementation of the Unit of Work pattern.
// The Unit of Work pattern maintains a list of objects affected by a business
// transaction and coordinates writing out changes and resolving concurrency problems.
//
// Key Features:
//   - Transaction management across multiple repositories
//   - Aggregate tracking for domain event processing
//   - Proper isolation between concurrent operations
//   - Automatic rollback on transaction failures
//   - Repository factory pattern for consistent database connections
//
// Usage Patterns:
//
// Basic Transaction Management:
//
//	factory := NewGormUnitOfWorkFactory(db)
//	uow := factory.Create()
//
//	if err := uow.Begin(ctx); err != nil {
//	    return err
//	}
//	defer func() {
//	    if r := recover(); r != nil {
//	        uow.Rollback(ctx)
//	        panic(r)
//	    }
//	}()
//
//	// Perform repository operations
//	if err := uow.OrderRepository().Add(ctx, order); err != nil {
//	    uow.Rollback(ctx)
//	    return err
//	}
//
//	return uow.Commit(ctx)
//
// Multi-Repository Transactions:
//
//	uow := factory.Create()
//	if err := uow.Begin(ctx); err != nil {
//	    return err
//	}
//
//	// All operations within same transaction
//	if err := uow.OrderRepository().Add(ctx, order); err != nil {
//	    uow.Rollback(ctx)
//	    return err
//	}
//
//	if err := uow.CourierRepository().Update(ctx, courier); err != nil {
//	    uow.Rollback(ctx)
//	    return err
//	}
//
//	return uow.Commit(ctx)
//
// Error Handling Best Practices:
//   - Always handle Begin() errors
//   - Use defer/recover for automatic rollback
//   - Explicit rollback on business logic errors
//   - Check commit errors for transaction conflicts
//
// Concurrency Considerations:
//   - Each UnitOfWork instance provides isolated transactions
//   - Multiple goroutines should use separate UnitOfWork instances
//   - Database-level locking may be needed for high-contention scenarios
//
// Performance Considerations:
//   - Keep transactions short to reduce lock contention
//   - Batch related operations within single transactions
//   - Use repository patterns to minimize database round trips
package postgres

import (
	"context"

	"delivery/internal/adapters/out/postgres/courierrepo"
	"delivery/internal/adapters/out/postgres/orderrepo"
	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/core/ports"

	"gorm.io/gorm"
)

// trackedAggregate represents an aggregate modified during the unit of work.
// This is useful for implementing patterns like event sourcing or outbox pattern.
type trackedAggregate struct {
	ID        kernel.UUID
	Aggregate interface{} // Will be changed to a common Aggregate interface in the future
}

// GormUnitOfWorkFactory creates UnitOfWork instances using GORM database connections.
// Factory ensures each business operation gets a fresh unit of work instance
// with proper isolation from other concurrent operations.
//
// Example:
//
//	db := setupGormDB() // your GORM database setup
//	factory := NewGormUnitOfWorkFactory(db)
//	uow := factory.Create()
//	defer func() {
//	    if err := recover(); err != nil {
//	        uow.Rollback(ctx)
//	        panic(err)
//	    }
//	}()
type GormUnitOfWorkFactory struct {
	db *gorm.DB
}

// NewGormUnitOfWorkFactory creates a factory for GORM-based unit of work instances.
// The provided database connection will be used for all created unit of work instances.
//
// Example:
//
//	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
//	if err != nil {
//	    log.Fatal("failed to connect database")
//	}
//	factory := NewGormUnitOfWorkFactory(db)
func NewGormUnitOfWorkFactory(db *gorm.DB) *GormUnitOfWorkFactory {
	return &GormUnitOfWorkFactory{db: db}
}

// Create produces a new UnitOfWork instance ready for business transaction management.
// Each instance maintains its own transaction state and aggregate tracking,
// ensuring proper isolation between concurrent operations.
//
// Example:
//
//	factory := NewGormUnitOfWorkFactory(db)
//	uow := factory.Create()
//
//	// Use the unit of work
//	if err := uow.Begin(ctx); err != nil {
//	    return err
//	}
//
//	// Perform repository operations
//	err := uow.OrderRepository().Add(ctx, order)
//	if err != nil {
//	    uow.Rollback(ctx)
//	    return err
//	}
//
//	return uow.Commit(ctx)
func (f *GormUnitOfWorkFactory) Create() ports.UnitOfWork {
	return &GormUnitOfWork{
		db:                f.db,
		trackedAggregates: make([]trackedAggregate, 0),
	}
}

// GormUnitOfWork coordinates database transactions and tracks aggregate changes
// for business operations. Implements the Unit of Work pattern using GORM's
// transaction capabilities to ensure data consistency and proper rollback handling.
//
// The unit of work tracks all aggregates modified during the transaction,
// enabling patterns like domain event publishing after successful commits
// or implementing the outbox pattern for reliable event processing.
//
// Example usage:
//
//	uow := factory.Create()
//
//	if err := uow.Begin(ctx); err != nil {
//	    return fmt.Errorf("failed to begin transaction: %w", err)
//	}
//
//	// Perform multiple repository operations
//	order := createNewOrder()
//	if err := uow.OrderRepository().Add(ctx, order); err != nil {
//	    uow.Rollback(ctx)
//	    return fmt.Errorf("failed to add order: %w", err)
//	}
//
//	courier := assignCourier()
//	if err := uow.CourierRepository().Update(ctx, courier); err != nil {
//	    uow.Rollback(ctx)
//	    return fmt.Errorf("failed to update courier: %w", err)
//	}
//
//	if err := uow.Commit(ctx); err != nil {
//	    return fmt.Errorf("failed to commit transaction: %w", err)
//	}
//
//	// Process tracked aggregates for domain events
//	for _, tracked := range uow.GetTrackedAggregates() {
//	    publishDomainEvents(tracked.Aggregate)
//	}
type GormUnitOfWork struct {
	db                *gorm.DB
	tx                *gorm.DB
	trackedAggregates []trackedAggregate
}

// Begin initiates a new database transaction for the unit of work.
// Subsequent repository operations will execute within this transaction context.
// Multiple calls to Begin on the same instance are safe and will not create nested transactions.
//
// Example:
//
//	uow := factory.Create()
//	if err := uow.Begin(ctx); err != nil {
//	    return fmt.Errorf("failed to begin transaction: %w", err)
//	}
func (uow *GormUnitOfWork) Begin(ctx context.Context) error {
	if uow.tx != nil {
		return nil
	}

	uow.tx = uow.db.WithContext(ctx).Begin()
	if uow.tx.Error != nil {
		return uow.tx.Error
	}

	return nil
}

// Commit finalizes all changes made within the current transaction.
// All tracked aggregates and their modifications become permanent in the database.
// After commit, the transaction is closed and cannot be reused.
//
// Returns error if no active transaction exists or if the commit operation fails.
//
// Example:
//
//	// Perform repository operations within transaction
//	err := uow.OrderRepository().Add(ctx, order)
//	if err != nil {
//	    uow.Rollback(ctx)
//	    return err
//	}
//
//	if err := uow.Commit(ctx); err != nil {
//	    return fmt.Errorf("failed to commit changes: %w", err)
//	}
func (uow *GormUnitOfWork) Commit(_ context.Context) error {
	if uow.tx == nil {
		return gorm.ErrInvalidTransaction
	}

	err := uow.tx.Commit().Error
	uow.tx = nil
	return err
}

// Rollback discards all changes made within the current transaction.
// Database returns to its state before the transaction began.
// After rollback, the transaction is closed and cannot be reused.
//
// Returns error if no active transaction exists or if the rollback operation fails.
//
// Example:
//
//	uow := factory.Create()
//	if err := uow.Begin(ctx); err != nil {
//	    return err
//	}
//
//	defer func() {
//	    if r := recover(); r != nil {
//	        uow.Rollback(ctx)
//	        panic(r)
//	    }
//	}()
//
//	// If any operation fails, rollback the transaction
//	if err := uow.OrderRepository().Add(ctx, order); err != nil {
//	    uow.Rollback(ctx)
//	    return err
//	}
func (uow *GormUnitOfWork) Rollback(_ context.Context) error {
	if uow.tx == nil {
		return gorm.ErrInvalidTransaction
	}

	err := uow.tx.Rollback().Error
	uow.tx = nil
	return err
}

// CourierRepository provides access to courier persistence operations within the unit of work.
// Repository operations will execute within the current transaction if one is active,
// otherwise they use the main database connection for immediate execution.
//
// The returned repository automatically tracks all courier aggregates that are
// added or updated, making them available via GetTrackedAggregates().
//
// Example:
//
//	uow := factory.Create()
//	uow.Begin(ctx)
//
//	courier := createNewCourier()
//	err := uow.CourierRepository().Add(ctx, courier)
//	if err != nil {
//	    uow.Rollback(ctx)
//	    return err
//	}
//
//	uow.Commit(ctx)
func (uow *GormUnitOfWork) CourierRepository() ports.CourierRepository {
	db := uow.db
	if uow.tx != nil {
		db = uow.tx
	}
	return courierrepo.NewGormCourierRepository(db, uow)
}

// OrderRepository provides access to order persistence operations within the unit of work.
// Repository operations will execute within the current transaction if one is active,
// otherwise they use the main database connection for immediate execution.
//
// The returned repository automatically tracks all order aggregates that are
// added or updated, making them available via GetTrackedAggregates().
//
// Example:
//
//	uow := factory.Create()
//	uow.Begin(ctx)
//
//	order := createNewOrder()
//	err := uow.OrderRepository().Add(ctx, order)
//	if err != nil {
//	    uow.Rollback(ctx)
//	    return err
//	}
//
//	uow.Commit(ctx)
func (uow *GormUnitOfWork) OrderRepository() ports.OrderRepository {
	db := uow.db
	if uow.tx != nil {
		db = uow.tx
	}
	return orderrepo.NewGormOrderRepository(db, uow)
}

// TrackAggregate registers a domain aggregate as modified within this unit of work.
// This method is typically called by repository implementations when aggregates
// are added, updated, or otherwise modified.
//
// The tracked aggregates can be retrieved via GetTrackedAggregates() after
// the transaction completes, enabling domain event processing or other
// post-transaction activities.
//
// Example (typically used by repository implementations):
//
//	func (r *GormOrderRepository) Add(ctx context.Context, order *order.Order) error {
//	    // Save to database
//	    if err := r.db.Create(orderDTO).Error; err != nil {
//	        return err
//	    }
//
//	    // Track the aggregate for post-transaction processing
//	    r.tracker.TrackAggregate(order.ID(), order)
//	    return nil
//	}
func (uow *GormUnitOfWork) TrackAggregate(id kernel.UUID, aggregate interface{}) {
	uow.trackedAggregates = append(uow.trackedAggregates, trackedAggregate{
		ID:        id,
		Aggregate: aggregate,
	})
}
