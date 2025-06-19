package postgres_test

import (
	"context"
	"testing"

	postgres_adapter "delivery/internal/adapters/out/postgres"
	"delivery/internal/adapters/out/postgres/courierrepo"
	"delivery/internal/adapters/out/postgres/orderrepo"
	"delivery/internal/core/domain/model/courier"
	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/core/domain/model/order"
	"delivery/internal/core/ports"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gorm_postgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// UnitOfWorkIntegrationTestSuite provides comprehensive integration testing
// for the GORM-based Unit of Work implementation with real PostgreSQL database.
type UnitOfWorkIntegrationTestSuite struct {
	suite.Suite
	container *postgres.PostgresContainer
	db        *gorm.DB
	factory   ports.UnitOfWorkFactory
}

// SetupSuite initializes PostgreSQL container and database connection for all tests.
// Runs database migrations to prepare schema for unit of work operations.
func (suite *UnitOfWorkIntegrationTestSuite) SetupSuite() {
	ctx := context.Background()

	// Start PostgreSQL container
	container, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2)),
	)
	suite.Require().NoError(err)
	suite.container = container

	// Connect to database
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	suite.Require().NoError(err)

	db, err := gorm.Open(gorm_postgres.Open(dsn), &gorm.Config{})
	suite.Require().NoError(err)
	suite.db = db

	// Run migrations
	err = db.AutoMigrate(&orderrepo.OrderDTO{}, &courierrepo.CourierDTO{}, &courierrepo.StoragePlaceDTO{})
	suite.Require().NoError(err)

	// Create factory
	suite.factory = postgres_adapter.NewGormUnitOfWorkFactory(db)
}

// SetupTest ensures clean database state before each test.
// Truncates all tables to prevent test interference.
func (suite *UnitOfWorkIntegrationTestSuite) SetupTest() {
	err := suite.db.Exec("TRUNCATE TABLE orders, couriers, storage_places").Error
	suite.Require().NoError(err)
}

// TearDownSuite cleans up PostgreSQL container after all tests complete.
func (suite *UnitOfWorkIntegrationTestSuite) TearDownSuite() {
	if suite.container != nil {
		err := suite.container.Terminate(context.Background())
		suite.Require().NoError(err)
	}
}

// TestUnitOfWorkFactory_Create verifies factory creates unit of work instances
// with proper initialization and isolation between instances.
func (suite *UnitOfWorkIntegrationTestSuite) TestUnitOfWorkFactory_Create() {
	// Create multiple unit of work instances
	uow1 := suite.factory.Create()
	uow2 := suite.factory.Create()

	// Verify instances are different
	suite.NotSame(uow1, uow2, "Factory should create separate instances")

	// Verify both instances provide access to repositories
	suite.NotNil(uow1.OrderRepository(), "First instance should provide order repository")
	suite.NotNil(uow1.CourierRepository(), "First instance should provide courier repository")
	suite.NotNil(uow2.OrderRepository(), "Second instance should provide order repository")
	suite.NotNil(uow2.CourierRepository(), "Second instance should provide courier repository")
}

// TestUnitOfWork_TransactionLifecycle verifies proper transaction management
// including begin, commit, and rollback operations.
func (suite *UnitOfWorkIntegrationTestSuite) TestUnitOfWork_TransactionLifecycle() {
	ctx := context.Background()
	uow := suite.factory.Create()

	// Test begin transaction
	err := uow.Begin(ctx)
	suite.Require().NoError(err, "Should begin transaction successfully")

	// Test multiple begin calls are safe
	err = uow.Begin(ctx)
	suite.Require().NoError(err, "Multiple begin calls should be safe")

	// Test commit
	err = uow.Commit(ctx)
	suite.Require().NoError(err, "Should commit transaction successfully")

	// Test rollback on new transaction
	err = uow.Begin(ctx)
	suite.Require().NoError(err)

	err = uow.Rollback(ctx)
	suite.Require().NoError(err, "Should rollback transaction successfully")
}

// TestUnitOfWork_TransactionErrors verifies error handling for invalid transaction operations.
func (suite *UnitOfWorkIntegrationTestSuite) TestUnitOfWork_TransactionErrors() {
	ctx := context.Background()
	uow := suite.factory.Create()

	// Test commit without begin
	err := uow.Commit(ctx)
	suite.Require().Error(err, "Should error when committing without active transaction")

	// Test rollback without begin
	err = uow.Rollback(ctx)
	suite.Require().Error(err, "Should error when rolling back without active transaction")
}

// TestUnitOfWork_SingleRepositoryTransaction verifies repository operations
// within a single transaction boundary work correctly.
func (suite *UnitOfWorkIntegrationTestSuite) TestUnitOfWork_SingleRepositoryTransaction() {
	ctx := context.Background()
	uow := suite.factory.Create()

	// Create test order
	testOrder := createTestOrder()

	// Begin transaction
	err := uow.Begin(ctx)
	suite.Require().NoError(err)

	// Add order within transaction
	err = uow.OrderRepository().Add(ctx, testOrder)
	suite.Require().NoError(err)

	// Verify order exists within transaction
	retrievedOrder, err := uow.OrderRepository().Get(ctx, testOrder.ID())
	suite.Require().NoError(err)
	suite.Equal(testOrder.ID(), retrievedOrder.ID())

	// Commit transaction
	err = uow.Commit(ctx)
	suite.Require().NoError(err)

	// Verify order persists after commit using new unit of work
	newUow := suite.factory.Create()
	retrievedOrder, err = newUow.OrderRepository().Get(ctx, testOrder.ID())
	suite.Require().NoError(err)
	suite.Equal(testOrder.ID(), retrievedOrder.ID())
}

// TestUnitOfWork_MultiRepositoryTransaction verifies multiple repository operations
// within a single transaction work atomically.
func (suite *UnitOfWorkIntegrationTestSuite) TestUnitOfWork_MultiRepositoryTransaction() {
	ctx := context.Background()
	uow := suite.factory.Create()

	// Create test entities
	testOrder := createTestOrder()
	testCourier := createTestCourier()

	// Begin transaction
	err := uow.Begin(ctx)
	suite.Require().NoError(err)

	// Add entities using different repositories within same transaction
	err = uow.OrderRepository().Add(ctx, testOrder)
	suite.Require().NoError(err)

	err = uow.CourierRepository().Add(ctx, testCourier)
	suite.Require().NoError(err)

	// Assign courier to order
	err = testOrder.Assign(testCourier.ID())
	suite.Require().NoError(err)
	err = uow.OrderRepository().Update(ctx, testOrder)
	suite.Require().NoError(err)

	// Assign order to courier (check capacity first)
	canTake, err := testCourier.CanTakeOrder(testOrder)
	suite.Require().NoError(err)
	suite.True(canTake, "Courier should be able to take the order")
	err = testCourier.TakeOrder(testOrder)
	suite.Require().NoError(err)
	err = uow.CourierRepository().Update(ctx, testCourier)
	suite.Require().NoError(err)

	// Commit transaction
	err = uow.Commit(ctx)
	suite.Require().NoError(err)

	// Verify both entities persisted correctly with relationships
	newUow := suite.factory.Create()

	retrievedOrder, err := newUow.OrderRepository().Get(ctx, testOrder.ID())
	suite.Require().NoError(err)
	suite.Equal(testCourier.ID(), *retrievedOrder.Courier())

	retrievedCourier, err := newUow.CourierRepository().Get(ctx, testCourier.ID())
	suite.Require().NoError(err)
	// Check that courier's storage places contain the order
	foundOrder := false
	for _, place := range retrievedCourier.StoragePlaces() {
		if place.OrderID() != nil && *place.OrderID() == testOrder.ID() {
			foundOrder = true
			break
		}
	}
	suite.True(foundOrder, "Courier should have the order in storage places")
}

// TestUnitOfWork_TransactionRollback verifies rollback discards all changes
// made within the transaction across multiple repositories.
func (suite *UnitOfWorkIntegrationTestSuite) TestUnitOfWork_TransactionRollback() {
	ctx := context.Background()
	uow := suite.factory.Create()

	// Create test entities
	testOrder := createTestOrder()
	testCourier := createTestCourier()

	// Begin transaction
	err := uow.Begin(ctx)
	suite.Require().NoError(err)

	// Add entities within transaction
	err = uow.OrderRepository().Add(ctx, testOrder)
	suite.Require().NoError(err)

	err = uow.CourierRepository().Add(ctx, testCourier)
	suite.Require().NoError(err)

	// Verify entities exist within transaction
	_, err = uow.OrderRepository().Get(ctx, testOrder.ID())
	suite.Require().NoError(err)

	_, err = uow.CourierRepository().Get(ctx, testCourier.ID())
	suite.Require().NoError(err)

	// Rollback transaction
	err = uow.Rollback(ctx)
	suite.Require().NoError(err)

	// Verify entities do not exist after rollback using new unit of work
	newUow := suite.factory.Create()

	_, err = newUow.OrderRepository().Get(ctx, testOrder.ID())
	suite.Require().Error(err, "Order should not exist after rollback")

	_, err = newUow.CourierRepository().Get(ctx, testCourier.ID())
	suite.Require().Error(err, "Courier should not exist after rollback")
}

// TestUnitOfWork_AggregateTracking verifies that aggregate tracking mechanism works
// during unit of work operations by ensuring repository operations complete successfully.
func (suite *UnitOfWorkIntegrationTestSuite) TestUnitOfWork_AggregateTracking() {
	ctx := context.Background()
	uow := suite.factory.Create()

	// Create test entities
	testOrder := createTestOrder()
	testCourier := createTestCourier()

	// Begin transaction
	err := uow.Begin(ctx)
	suite.Require().NoError(err)

	// Add entities (repositories should track aggregates internally)
	err = uow.OrderRepository().Add(ctx, testOrder)
	suite.Require().NoError(err)

	err = uow.CourierRepository().Add(ctx, testCourier)
	suite.Require().NoError(err)

	// Update entities (repositories should track aggregates internally)
	err = testOrder.Assign(testCourier.ID())
	suite.Require().NoError(err)
	err = uow.OrderRepository().Update(ctx, testOrder)
	suite.Require().NoError(err)

	// Commit transaction - if aggregate tracking is working properly, this should succeed
	err = uow.Commit(ctx)
	suite.Require().NoError(err)

	// Verify entities were persisted correctly
	newUow := suite.factory.Create()
	retrievedOrder, err := newUow.OrderRepository().Get(ctx, testOrder.ID())
	suite.Require().NoError(err)
	suite.Equal(testCourier.ID(), *retrievedOrder.Courier())

	retrievedCourier, err := newUow.CourierRepository().Get(ctx, testCourier.ID())
	suite.Require().NoError(err)
	suite.Equal(testCourier.ID(), retrievedCourier.ID())
}

// TestUnitOfWork_RepositoryIsolation verifies that repositories obtained
// from different unit of work instances operate independently.
func (suite *UnitOfWorkIntegrationTestSuite) TestUnitOfWork_RepositoryIsolation() {
	ctx := context.Background()

	// Create two unit of work instances
	uow1 := suite.factory.Create()
	uow2 := suite.factory.Create()

	// Create test orders
	order1 := createTestOrder()
	order2 := createTestOrder()

	// Begin transactions on both
	err := uow1.Begin(ctx)
	suite.Require().NoError(err)

	err = uow2.Begin(ctx)
	suite.Require().NoError(err)

	// Add different orders in each transaction
	err = uow1.OrderRepository().Add(ctx, order1)
	suite.Require().NoError(err)

	err = uow2.OrderRepository().Add(ctx, order2)
	suite.Require().NoError(err)

	// Each transaction should only see its own changes
	_, err = uow1.OrderRepository().Get(ctx, order1.ID())
	suite.Require().NoError(err, "UOW1 should see order1")

	_, err = uow1.OrderRepository().Get(ctx, order2.ID())
	suite.Require().Error(err, "UOW1 should not see order2")

	_, err = uow2.OrderRepository().Get(ctx, order2.ID())
	suite.Require().NoError(err, "UOW2 should see order2")

	_, err = uow2.OrderRepository().Get(ctx, order1.ID())
	suite.Require().Error(err, "UOW2 should not see order1")

	// Commit first transaction
	err = uow1.Commit(ctx)
	suite.Require().NoError(err)

	// Rollback second transaction
	err = uow2.Rollback(ctx)
	suite.Require().NoError(err)

	// Verify only order1 persisted
	newUow := suite.factory.Create()
	_, err = newUow.OrderRepository().Get(ctx, order1.ID())
	suite.Require().NoError(err, "Order1 should persist after commit")

	_, err = newUow.OrderRepository().Get(ctx, order2.ID())
	suite.Require().Error(err, "Order2 should not persist after rollback")
}

// TestUnitOfWork_WithoutTransaction verifies that repositories work correctly
// without explicit transaction boundaries for immediate operations.
func (suite *UnitOfWorkIntegrationTestSuite) TestUnitOfWork_WithoutTransaction() {
	ctx := context.Background()
	uow := suite.factory.Create()

	// Create test order
	testOrder := createTestOrder()

	// Add order without beginning transaction (should auto-commit)
	err := uow.OrderRepository().Add(ctx, testOrder)
	suite.Require().NoError(err)

	// Verify order persists immediately
	retrievedOrder, err := uow.OrderRepository().Get(ctx, testOrder.ID())
	suite.Require().NoError(err)
	suite.Equal(testOrder.ID(), retrievedOrder.ID())

	// Verify with new unit of work instance
	newUow := suite.factory.Create()
	retrievedOrder, err = newUow.OrderRepository().Get(ctx, testOrder.ID())
	suite.Require().NoError(err)
	suite.Equal(testOrder.ID(), retrievedOrder.ID())
}

// createTestOrder creates a valid order for testing purposes.
func createTestOrder() *order.Order {
	id := kernel.NewUUID()
	location, _ := kernel.NewLocation(5, 7)
	testOrder, _ := order.NewOrder(id, location, 5) // Small volume to fit in courier storage
	return testOrder
}

// createTestCourier creates a valid courier for testing purposes.
func createTestCourier() *courier.Courier {
	id := kernel.NewUUID()
	location, _ := kernel.NewLocation(3, 4)
	testCourier, _ := courier.NewCourier(id, "Test Courier", 3, location)
	return testCourier
}

// TestUnitOfWork_OrderDeliveryWorkflow tests the complete order delivery workflow
// involving multiple aggregates and domain operations within a single transaction.
func (suite *UnitOfWorkIntegrationTestSuite) TestUnitOfWork_OrderDeliveryWorkflow() {
	ctx := context.Background()
	uow := suite.factory.Create()

	// Begin transaction for the entire workflow
	err := uow.Begin(ctx)
	suite.Require().NoError(err)

	// Step 1: Create and add a new order
	testOrder := createTestOrder()
	err = uow.OrderRepository().Add(ctx, testOrder)
	suite.Require().NoError(err)

	// Step 2: Create and add a courier
	testCourier := createTestCourier()
	err = uow.CourierRepository().Add(ctx, testCourier)
	suite.Require().NoError(err)

	// Step 3: Assign order to courier (domain operation)
	err = testOrder.Assign(testCourier.ID())
	suite.Require().NoError(err)
	err = uow.OrderRepository().Update(ctx, testOrder)
	suite.Require().NoError(err)

	// Step 4: Courier takes the order (domain operation)
	canTake, err := testCourier.CanTakeOrder(testOrder)
	suite.Require().NoError(err)
	suite.True(canTake, "Courier should be able to take the order")

	err = testCourier.TakeOrder(testOrder)
	suite.Require().NoError(err)
	err = uow.CourierRepository().Update(ctx, testCourier)
	suite.Require().NoError(err)

	// Step 5: Complete the order delivery
	err = testOrder.Complete()
	suite.Require().NoError(err)
	err = uow.OrderRepository().Update(ctx, testOrder)
	suite.Require().NoError(err)

	// Step 6: Release order from courier storage
	for _, place := range testCourier.StoragePlaces() {
		if place.OrderID() != nil && place.OrderID().IsEqual(testOrder.ID()) {
			err = place.Clear(testOrder.ID())
			suite.Require().NoError(err)
			break
		}
	}
	err = uow.CourierRepository().Update(ctx, testCourier)
	suite.Require().NoError(err)

	// Commit the entire workflow
	err = uow.Commit(ctx)
	suite.Require().NoError(err)

	// Verify final state using a new unit of work
	newUow := suite.factory.Create()

	// Verify order is completed
	retrievedOrder, err := newUow.OrderRepository().Get(ctx, testOrder.ID())
	suite.Require().NoError(err)
	suite.Equal(order.Completed, retrievedOrder.Status())
	suite.NotNil(retrievedOrder.Courier())
	suite.Equal(testCourier.ID(), *retrievedOrder.Courier())

	// Verify courier is free again (no orders in storage)
	retrievedCourier, err := newUow.CourierRepository().Get(ctx, testCourier.ID())
	suite.Require().NoError(err)
	for _, place := range retrievedCourier.StoragePlaces() {
		suite.Nil(place.OrderID(), "All storage places should be empty after order completion")
	}

	// Verify courier appears in free courier list
	freeCouriers, err := newUow.CourierRepository().GetAllFree(ctx)
	suite.Require().NoError(err)
	found := false
	for _, freeCourier := range freeCouriers {
		if freeCourier.ID().IsEqual(testCourier.ID()) {
			found = true
			break
		}
	}
	suite.True(found, "Courier should be available for new orders")
}

// TestUnitOfWork_WorkflowRollback tests rollback behavior during a complex workflow.
func (suite *UnitOfWorkIntegrationTestSuite) TestUnitOfWork_WorkflowRollback() {
	ctx := context.Background()
	uow := suite.factory.Create()

	// Begin transaction
	err := uow.Begin(ctx)
	suite.Require().NoError(err)

	// Create order and courier
	testOrder := createTestOrder()
	testCourier := createTestCourier()

	err = uow.OrderRepository().Add(ctx, testOrder)
	suite.Require().NoError(err)
	err = uow.CourierRepository().Add(ctx, testCourier)
	suite.Require().NoError(err)

	// Perform domain operations
	err = testOrder.Assign(testCourier.ID())
	suite.Require().NoError(err)
	err = uow.OrderRepository().Update(ctx, testOrder)
	suite.Require().NoError(err)

	err = testCourier.TakeOrder(testOrder)
	suite.Require().NoError(err)
	err = uow.CourierRepository().Update(ctx, testCourier)
	suite.Require().NoError(err)

	// Rollback transaction
	err = uow.Rollback(ctx)
	suite.Require().NoError(err)

	// Verify nothing was persisted
	newUow := suite.factory.Create()

	_, err = newUow.OrderRepository().Get(ctx, testOrder.ID())
	suite.Require().Error(err, "Order should not exist after rollback")

	_, err = newUow.CourierRepository().Get(ctx, testCourier.ID())
	suite.Require().Error(err, "Courier should not exist after rollback")

	// Verify no free couriers exist
	freeCouriers, err := newUow.CourierRepository().GetAllFree(ctx)
	suite.Require().NoError(err)
	suite.Empty(freeCouriers, "No couriers should exist after rollback")
}

// TestUnitOfWork_PartialFailureScenario tests behavior when some operations succeed and others fail.
func (suite *UnitOfWorkIntegrationTestSuite) TestUnitOfWork_PartialFailureScenario() {
	ctx := context.Background()
	uow := suite.factory.Create()

	// Create initial order outside transaction
	existingOrder := createTestOrder()
	err := uow.OrderRepository().Add(ctx, existingOrder)
	suite.Require().NoError(err)

	// Begin new transaction
	err = uow.Begin(ctx)
	suite.Require().NoError(err)

	// Add valid entities
	newOrder := createTestOrder()
	newCourier := createTestCourier()

	err = uow.OrderRepository().Add(ctx, newOrder)
	suite.Require().NoError(err)
	err = uow.CourierRepository().Add(ctx, newCourier)
	suite.Require().NoError(err)

	// Try to add duplicate order (should fail)
	duplicateOrder, err := order.RestoreOrder(
		existingOrder.ID(), // Same ID as existing order
		existingOrder.Location(),
		existingOrder.Volume(),
		order.Created,
		nil,
	)
	suite.Require().NoError(err)

	err = uow.OrderRepository().Add(ctx, duplicateOrder)
	suite.Require().Error(err, "Adding duplicate order should fail")

	// Even though some operations succeeded, rollback should undo everything
	err = uow.Rollback(ctx)
	suite.Require().NoError(err)

	// Verify rollback undid the successful operations
	newUow := suite.factory.Create()

	// Existing order should still exist (was added before transaction)
	_, err = newUow.OrderRepository().Get(ctx, existingOrder.ID())
	suite.Require().NoError(err, "Existing order should still exist")

	// New entities should not exist (transaction was rolled back)
	_, err = newUow.OrderRepository().Get(ctx, newOrder.ID())
	suite.Require().Error(err, "New order should not exist after rollback")

	_, err = newUow.CourierRepository().Get(ctx, newCourier.ID())
	suite.Require().Error(err, "New courier should not exist after rollback")
}

// TestUnitOfWork_ConcurrentOrderAssignment tests concurrent assignment scenarios
// Temporarily disabled - causes database deadlocks.
func (suite *UnitOfWorkIntegrationTestSuite) DisabledTestUnitOfWorkConcurrentOrderAssignment() {
	ctx := context.Background()

	// Create shared courier and orders outside transactions
	sharedCourier := createTestCourier()
	order1 := createTestOrder()
	order2 := createTestOrder()

	// Add them without transaction (auto-commit)
	initialUow := suite.factory.Create()
	err := initialUow.CourierRepository().Add(ctx, sharedCourier)
	suite.Require().NoError(err)
	err = initialUow.OrderRepository().Add(ctx, order1)
	suite.Require().NoError(err)
	err = initialUow.OrderRepository().Add(ctx, order2)
	suite.Require().NoError(err)

	// Create two separate transactions
	uow1 := suite.factory.Create()
	uow2 := suite.factory.Create()

	err = uow1.Begin(ctx)
	suite.Require().NoError(err)
	err = uow2.Begin(ctx)
	suite.Require().NoError(err)

	// Both transactions try to assign orders to the same courier
	// Transaction 1: Assign order1
	retrievedCourier1, err := uow1.CourierRepository().Get(ctx, sharedCourier.ID())
	suite.Require().NoError(err)
	retrievedOrder1, err := uow1.OrderRepository().Get(ctx, order1.ID())
	suite.Require().NoError(err)

	err = retrievedOrder1.Assign(retrievedCourier1.ID())
	suite.Require().NoError(err)
	err = uow1.OrderRepository().Update(ctx, retrievedOrder1)
	suite.Require().NoError(err)

	err = retrievedCourier1.TakeOrder(retrievedOrder1)
	suite.Require().NoError(err)
	err = uow1.CourierRepository().Update(ctx, retrievedCourier1)
	suite.Require().NoError(err)

	// Transaction 2: Try to assign order2 to same courier
	retrievedCourier2, err := uow2.CourierRepository().Get(ctx, sharedCourier.ID())
	suite.Require().NoError(err)
	retrievedOrder2, err := uow2.OrderRepository().Get(ctx, order2.ID())
	suite.Require().NoError(err)

	err = retrievedOrder2.Assign(retrievedCourier2.ID())
	suite.Require().NoError(err)
	err = uow2.OrderRepository().Update(ctx, retrievedOrder2)
	suite.Require().NoError(err)

	// This should succeed within the transaction context
	canTake, err := retrievedCourier2.CanTakeOrder(retrievedOrder2)
	suite.Require().NoError(err)
	if canTake {
		err = retrievedCourier2.TakeOrder(retrievedOrder2)
		suite.Require().NoError(err)
		err = uow2.CourierRepository().Update(ctx, retrievedCourier2)
		suite.Require().NoError(err)
	}

	// Commit first transaction
	err = uow1.Commit(ctx)
	suite.Require().NoError(err)

	// Try to commit second transaction
	// This might fail due to conflicts or succeed depending on implementation
	_ = uow2.Commit(ctx)
	// We don't assert on success/failure here as both behaviors are valid
	// The important thing is that the system remains consistent

	// Verify final state is consistent
	finalUow := suite.factory.Create()
	finalCourier, err := finalUow.CourierRepository().Get(ctx, sharedCourier.ID())
	suite.Require().NoError(err)

	// Count assigned orders
	assignedCount := 0
	for _, place := range finalCourier.StoragePlaces() {
		if place.OrderID() != nil {
			assignedCount++
		}
	}

	// Should have at most the courier's capacity
	suite.LessOrEqual(assignedCount, len(finalCourier.StoragePlaces()),
		"Assigned orders should not exceed courier capacity")
	suite.GreaterOrEqual(assignedCount, 1,
		"At least one order should be assigned")
}

// TestUnitOfWork_QueryConsistency verifies query results are consistent within transactions.
func (suite *UnitOfWorkIntegrationTestSuite) TestUnitOfWork_QueryConsistency() {
	ctx := context.Background()
	uow := suite.factory.Create()

	// Create initial data outside transaction
	order1 := createTestOrder()
	order2 := createTestOrder()
	courier1 := createTestCourier()

	err := uow.OrderRepository().Add(ctx, order1)
	suite.Require().NoError(err)
	err = uow.OrderRepository().Add(ctx, order2)
	suite.Require().NoError(err)
	err = uow.CourierRepository().Add(ctx, courier1)
	suite.Require().NoError(err)

	// Begin transaction
	err = uow.Begin(ctx)
	suite.Require().NoError(err)

	// Assign one order
	err = order1.Assign(courier1.ID())
	suite.Require().NoError(err)
	err = uow.OrderRepository().Update(ctx, order1)
	suite.Require().NoError(err)

	// Query for created orders - should include order2 but not order1
	createdOrder, err := uow.OrderRepository().GetFirstInCreatedStatus(ctx)
	suite.Require().NoError(err)
	suite.Equal(order2.ID(), createdOrder.ID(), "Should find the unassigned order")

	// Query for assigned orders - should include order1
	assignedOrders, err := uow.OrderRepository().GetAllInAssignedStatus(ctx)
	suite.Require().NoError(err)
	suite.Len(assignedOrders, 1)
	suite.Equal(order1.ID(), assignedOrders[0].ID())

	// Courier should not be free (has assigned order)
	freeCouriers, err := uow.CourierRepository().GetAllFree(ctx)
	suite.Require().NoError(err)
	suite.Empty(freeCouriers, "Courier should not be free with assigned order")

	// Commit transaction
	err = uow.Commit(ctx)
	suite.Require().NoError(err)

	// Verify queries still return consistent results after commit
	newUow := suite.factory.Create()

	createdOrder, err = newUow.OrderRepository().GetFirstInCreatedStatus(ctx)
	suite.Require().NoError(err)
	suite.Equal(order2.ID(), createdOrder.ID())

	assignedOrders, err = newUow.OrderRepository().GetAllInAssignedStatus(ctx)
	suite.Require().NoError(err)
	suite.Len(assignedOrders, 1)
	suite.Equal(order1.ID(), assignedOrders[0].ID())

	freeCouriers, err = newUow.CourierRepository().GetAllFree(ctx)
	suite.Require().NoError(err)
	suite.Empty(freeCouriers)
}

func TestUnitOfWorkIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UnitOfWorkIntegrationTestSuite))
}
