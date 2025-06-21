package courierrepo_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"delivery/internal/adapters/out/postgres/courierrepo"
	"delivery/internal/adapters/out/postgres/orderrepo"
	"delivery/internal/core/domain/model/courier"
	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/core/domain/model/order"
	"delivery/internal/pkg/errs"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// MockAggregateTracker is a mock implementation of aggregateTracker interface.
type MockAggregateTracker struct {
	mock.Mock
}

func (m *MockAggregateTracker) TrackAggregate(id kernel.UUID, aggregate interface{}) {
	m.Called(id, aggregate)
}

// CourierRepositoryIntegrationTestSuite provides integration tests for CourierRepository
// using PostgreSQL containers to verify database persistence behavior.
type CourierRepositoryIntegrationTestSuite struct {
	suite.Suite
	container         *postgres.PostgresContainer
	db                *gorm.DB
	courierRepository *courierrepo.GormCourierRepository
	orderRepository   *orderrepo.GormOrderRepository
	tracker           *MockAggregateTracker
}

func (suite *CourierRepositoryIntegrationTestSuite) SetupSuite() {
	ctx := context.Background()

	// Start PostgreSQL container
	container, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	suite.Require().NoError(err)
	suite.container = container

	// Get connection string and connect to database
	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	suite.Require().NoError(err)

	db, err := gorm.Open(postgresdriver.Open(connStr), &gorm.Config{})
	suite.Require().NoError(err)
	suite.db = db

	// Auto-migrate the schema
	suite.Require().NoError(db.AutoMigrate(
		&courierrepo.CourierDTO{},
		&courierrepo.StoragePlaceDTO{},
		&orderrepo.OrderDTO{},
	))
}

func (suite *CourierRepositoryIntegrationTestSuite) SetupTest() {
	// Clean the database before each test
	suite.Require().NoError(suite.db.Exec("TRUNCATE TABLE storage_places, couriers, orders").Error)

	// Create fresh repositories and tracker for each test
	suite.tracker = new(MockAggregateTracker)
	suite.courierRepository = courierrepo.NewGormCourierRepository(suite.db, suite.tracker)
	suite.orderRepository = orderrepo.NewGormOrderRepository(suite.db, suite.tracker)
}

func (suite *CourierRepositoryIntegrationTestSuite) TearDownSuite() {
	if suite.container != nil {
		suite.Require().NoError(suite.container.Terminate(context.Background()))
	}
}

func (suite *CourierRepositoryIntegrationTestSuite) TestAdd_ValidCourier_Success() {
	ctx := context.Background()

	// Create valid courier with storage places
	courier := suite.createTestCourier()

	// Set expectations on mock
	suite.tracker.On("TrackAggregate", courier.ID(), courier).Once()

	// Add courier to repository
	err := suite.courierRepository.Add(ctx, courier)
	suite.Require().NoError(err)

	// Verify courier was persisted
	suite.assertCourierCount(1)

	// Verify storage places were persisted
	suite.assertStoragePlaceCount(len(courier.StoragePlaces()))

	// Assert that all expectations were met
	suite.tracker.AssertExpectations(suite.T())
}

func (suite *CourierRepositoryIntegrationTestSuite) TestAdd_InvalidCourier_BusinessRules() {
	testCases := []struct {
		name     string
		setup    func() *courier.Courier
		expected string
	}{
		{
			name: "invalid speed",
			setup: func() *courier.Courier {
				id := kernel.NewUUID()
				location, _ := kernel.NewLocation(3, 7)
				storagePlaces, _ := suite.createTestStoragePlaces()
				c, _ := courier.RestoreCourier(id, "Test Courier", -1, location, storagePlaces)
				return c
			},
			expected: "speed",
		},
		{
			name: "empty name",
			setup: func() *courier.Courier {
				id := kernel.NewUUID()
				location, _ := kernel.NewLocation(3, 7)
				storagePlaces, _ := suite.createTestStoragePlaces()
				c, _ := courier.RestoreCourier(id, "", 5, location, storagePlaces)
				return c
			},
			expected: "name",
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create invalid courier
			invalidCourier := tc.setup()
			if invalidCourier == nil {
				// Constructor validation failed as expected
				return
			}

			// Try to add invalid courier
			err := suite.courierRepository.Add(ctx, invalidCourier)
			suite.Require().Error(err)
			suite.Contains(err.Error(), tc.expected)

			// Verify no courier was persisted
			suite.assertCourierCount(0)
			suite.tracker.AssertExpectations(suite.T())
		})
	}
}

func (suite *CourierRepositoryIntegrationTestSuite) TestGet_ExistingCourier_ReturnsCourierWithStoragePlaces() {
	ctx := context.Background()

	// Create and add courier
	originalCourier := suite.createTestCourier()

	// Set expectations for Add operation
	suite.tracker.On("TrackAggregate", originalCourier.ID(), originalCourier).Once()

	err := suite.courierRepository.Add(ctx, originalCourier)
	suite.Require().NoError(err)

	// Retrieve courier
	retrievedCourier, err := suite.courierRepository.Get(ctx, originalCourier.ID())
	suite.Require().NoError(err)

	// Verify courier details
	suite.Equal(originalCourier.ID(), retrievedCourier.ID())
	suite.Equal(originalCourier.Name(), retrievedCourier.Name())
	suite.Equal(originalCourier.Speed(), retrievedCourier.Speed())
	suite.Equal(originalCourier.Location().X(), retrievedCourier.Location().X())
	suite.Equal(originalCourier.Location().Y(), retrievedCourier.Location().Y())

	// Verify storage places were loaded
	suite.Len(retrievedCourier.StoragePlaces(), len(originalCourier.StoragePlaces()))
	for i, originalSP := range originalCourier.StoragePlaces() {
		retrievedSP := retrievedCourier.StoragePlaces()[i]
		suite.Equal(originalSP.ID(), retrievedSP.ID())
		suite.Equal(originalSP.Name(), retrievedSP.Name())
		suite.Equal(originalSP.TotalVolume(), retrievedSP.TotalVolume())
		suite.Equal(originalSP.OrderID(), retrievedSP.OrderID())
	}

	// Assert that all expectations were met
	suite.tracker.AssertExpectations(suite.T())
}

func (suite *CourierRepositoryIntegrationTestSuite) TestGet_NonExistentCourier_ReturnsNotFoundError() {
	ctx := context.Background()

	// Try to get non-existent courier
	nonExistentID := kernel.NewUUID()
	retrievedCourier, err := suite.courierRepository.Get(ctx, nonExistentID)

	// Verify error and result
	suite.Nil(retrievedCourier)
	suite.Require().Error(err)

	var notFoundErr *errs.ObjectNotFoundError
	suite.Require().ErrorAs(err, &notFoundErr)

	// Assert no unexpected calls
	suite.tracker.AssertExpectations(suite.T())
}

func (suite *CourierRepositoryIntegrationTestSuite) TestUpdate_CourierChanges() {
	testCases := []struct {
		name   string
		setup  func(*courier.Courier) *courier.Courier
		verify func(*courier.Courier, *courier.Courier)
	}{
		{
			name: "location change",
			setup: func(original *courier.Courier) *courier.Courier {
				newLocation, _ := kernel.NewLocation(8, 9)
				updated, _ := courier.RestoreCourier(
					original.ID(),
					original.Name(),
					original.Speed(),
					newLocation,
					original.StoragePlaces(),
				)
				return updated
			},
			verify: func(_, retrieved *courier.Courier) {
				suite.Equal(kernel.Coordinate(8), retrieved.Location().X())
				suite.Equal(kernel.Coordinate(9), retrieved.Location().Y())
			},
		},
		{
			name: "storage place order assignment",
			setup: func(original *courier.Courier) *courier.Courier {
				// Create order for storage place assignment
				order := suite.createTestOrderWithStatus(context.Background(), original.ID(), order.Assigned)

				// Mock order repository to add the order
				suite.tracker.On("TrackAggregate", order.ID(), order).Once()
				err := suite.orderRepository.Add(context.Background(), order)
				suite.Require().NoError(err)

				// Simulate taking the order (assign to storage place)
				canTake, err := original.CanTakeOrder(order)
				suite.Require().NoError(err)
				suite.True(canTake)

				err = original.TakeOrder(order)
				suite.Require().NoError(err)

				return original
			},
			verify: func(_, retrieved *courier.Courier) {
				// Verify storage place has order assigned
				foundAssignedPlace := false
				for _, place := range retrieved.StoragePlaces() {
					if place.OrderID() != nil {
						foundAssignedPlace = true
						break
					}
				}
				suite.True(foundAssignedPlace, "Should have at least one storage place with assigned order")
			},
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create and add initial courier
			originalCourier := suite.createTestCourier()
			suite.tracker.On("TrackAggregate", originalCourier.ID(), originalCourier).Once()
			err := suite.courierRepository.Add(ctx, originalCourier)
			suite.Require().NoError(err)

			// Apply changes
			updatedCourier := tc.setup(originalCourier)
			suite.tracker.On("TrackAggregate", updatedCourier.ID(), updatedCourier).Once()

			// Update courier in repository
			err = suite.courierRepository.Update(ctx, updatedCourier)
			suite.Require().NoError(err)

			// Retrieve and verify updated courier
			retrievedCourier, err := suite.courierRepository.Get(ctx, updatedCourier.ID())
			suite.Require().NoError(err)

			// Verify changes
			tc.verify(updatedCourier, retrievedCourier)

			suite.tracker.AssertExpectations(suite.T())
		})
	}
}

func (suite *CourierRepositoryIntegrationTestSuite) TestUpdate_NonExistentCourier_ReturnsError() {
	ctx := context.Background()

	// Create courier that doesn't exist in database
	nonExistentCourier := suite.createTestCourier()

	// No expectations on tracker since operation should fail

	// Try to update non-existent courier
	err := suite.courierRepository.Update(ctx, nonExistentCourier)
	suite.Require().Error(err)

	// Assert no unexpected calls
	suite.tracker.AssertExpectations(suite.T())
}

func (suite *CourierRepositoryIntegrationTestSuite) TestGetAllFree_NoCouriersAssigned_ReturnsAllCouriers() {
	ctx := context.Background()

	// Create and add multiple couriers
	courier1 := suite.createTestCourier()
	courier2 := suite.createTestCourierWithName("Courier 2")

	// Set expectations for both couriers
	suite.tracker.On("TrackAggregate", courier1.ID(), courier1).Once()
	suite.tracker.On("TrackAggregate", courier2.ID(), courier2).Once()

	err := suite.courierRepository.Add(ctx, courier1)
	suite.Require().NoError(err)

	err = suite.courierRepository.Add(ctx, courier2)
	suite.Require().NoError(err)

	// Get all free couriers
	freeCouriers, err := suite.courierRepository.GetAllFree(ctx)
	suite.Require().NoError(err)

	// Verify both couriers are returned as free
	suite.Len(freeCouriers, 2)

	// Assert that all expectations were met
	suite.tracker.AssertExpectations(suite.T())
}

func (suite *CourierRepositoryIntegrationTestSuite) TestGetAllFree_SomeCouriersAssigned_ReturnsOnlyFreeCouriers() {
	ctx := context.Background()

	// Create and add multiple couriers
	freeCourier := suite.createTestCourierWithName("Free Courier")
	assignedCourier := suite.createTestCourierWithName("Assigned Courier")

	// Set expectations for both couriers
	suite.tracker.On("TrackAggregate", freeCourier.ID(), freeCourier).Once()
	suite.tracker.On("TrackAggregate", assignedCourier.ID(), assignedCourier).Once()

	err := suite.courierRepository.Add(ctx, freeCourier)
	suite.Require().NoError(err)

	err = suite.courierRepository.Add(ctx, assignedCourier)
	suite.Require().NoError(err)

	// Create and add an order assigned to one courier
	assignedOrder := suite.createTestOrderAssignedToCourier(ctx, assignedCourier.ID())

	// Set expectations for order
	suite.tracker.On("TrackAggregate", assignedOrder.ID(), assignedOrder).Once()

	err = suite.orderRepository.Add(ctx, assignedOrder)
	suite.Require().NoError(err)

	// Get all free couriers
	freeCouriers, err := suite.courierRepository.GetAllFree(ctx)
	suite.Require().NoError(err)

	// Verify only the free courier is returned
	suite.Len(freeCouriers, 1)
	suite.Equal(freeCourier.ID(), freeCouriers[0].ID())
	suite.Equal("Free Courier", freeCouriers[0].Name())

	// Assert that all expectations were met
	suite.tracker.AssertExpectations(suite.T())
}

func (suite *CourierRepositoryIntegrationTestSuite) TestGetAllFree_AllCouriersAssigned_ReturnsEmptySlice() {
	ctx := context.Background()

	// Create and add courier
	assignedCourier := suite.createTestCourierWithName("Assigned Courier")

	// Set expectations for courier
	suite.tracker.On("TrackAggregate", assignedCourier.ID(), assignedCourier).Once()

	err := suite.courierRepository.Add(ctx, assignedCourier)
	suite.Require().NoError(err)

	// Create and add an order assigned to the courier
	assignedOrder := suite.createTestOrderAssignedToCourier(ctx, assignedCourier.ID())

	// Set expectations for order
	suite.tracker.On("TrackAggregate", assignedOrder.ID(), assignedOrder).Once()

	err = suite.orderRepository.Add(ctx, assignedOrder)
	suite.Require().NoError(err)

	// Get all free couriers
	freeCouriers, err := suite.courierRepository.GetAllFree(ctx)
	suite.Require().NoError(err)

	// Verify no couriers are returned
	suite.Empty(freeCouriers)

	// Assert that all expectations were met
	suite.tracker.AssertExpectations(suite.T())
}

func (suite *CourierRepositoryIntegrationTestSuite) TestGetAllFree_CourierWithCompletedOrder_ReturnsCourierAsFree() {
	ctx := context.Background()

	// Create and add courier
	courierWithCompletedOrder := suite.createTestCourierWithName("Courier With Completed Order")

	// Set expectations for courier
	suite.tracker.On("TrackAggregate", courierWithCompletedOrder.ID(), courierWithCompletedOrder).Once()

	err := suite.courierRepository.Add(ctx, courierWithCompletedOrder)
	suite.Require().NoError(err)

	// Create and add a completed order for the courier
	completedOrder := suite.createTestOrderWithStatus(ctx, courierWithCompletedOrder.ID(), order.Completed)

	// Set expectations for order
	suite.tracker.On("TrackAggregate", completedOrder.ID(), completedOrder).Once()

	err = suite.orderRepository.Add(ctx, completedOrder)
	suite.Require().NoError(err)

	// Get all free couriers
	freeCouriers, err := suite.courierRepository.GetAllFree(ctx)
	suite.Require().NoError(err)

	// Verify the courier with completed order is returned as free
	suite.Len(freeCouriers, 1)
	suite.Equal(courierWithCompletedOrder.ID(), freeCouriers[0].ID())
	suite.Equal("Courier With Completed Order", freeCouriers[0].Name())

	// Assert that all expectations were met
	suite.tracker.AssertExpectations(suite.T())
}

func (suite *CourierRepositoryIntegrationTestSuite) TestGetAllFree_CourierWithCreatedOrder_ReturnsCourierAsFree() {
	ctx := context.Background()

	// Create and add courier
	freeCourier := suite.createTestCourierWithName("Free Courier")

	// Set expectations for courier
	suite.tracker.On("TrackAggregate", freeCourier.ID(), freeCourier).Once()

	err := suite.courierRepository.Add(ctx, freeCourier)
	suite.Require().NoError(err)

	// Create and add an order in Created status (not assigned to any courier)
	createdOrder := suite.createTestOrderWithStatus(ctx, kernel.UUID{}, order.Created)

	// Set expectations for order
	suite.tracker.On("TrackAggregate", createdOrder.ID(), createdOrder).Once()

	err = suite.orderRepository.Add(ctx, createdOrder)
	suite.Require().NoError(err)

	// Get all free couriers
	freeCouriers, err := suite.courierRepository.GetAllFree(ctx)
	suite.Require().NoError(err)

	// Verify the courier is returned as free (Created orders don't assign couriers)
	suite.Len(freeCouriers, 1)
	suite.Equal(freeCourier.ID(), freeCouriers[0].ID())
	suite.Equal("Free Courier", freeCouriers[0].Name())

	// Assert that all expectations were met
	suite.tracker.AssertExpectations(suite.T())
}

func (suite *CourierRepositoryIntegrationTestSuite) TestGetAllFree_BusinessScenariosSimplified() {
	testCases := []struct {
		name               string
		courierStatusPairs []struct {
			courierName  string
			orderStatus  order.Status
			expectedFree bool
		}
		expectedFreeCount int
	}{
		{
			name: "mixed order statuses",
			courierStatusPairs: []struct {
				courierName  string
				orderStatus  order.Status
				expectedFree bool
			}{
				{"Free Courier", order.Created, true},        // Created orders don't assign couriers
				{"Assigned Courier", order.Assigned, false},  // Assigned orders make couriers busy
				{"Completed Courier", order.Completed, true}, // Completed orders free up couriers
			},
			expectedFreeCount: 2,
		},
		{
			name: "all assigned",
			courierStatusPairs: []struct {
				courierName  string
				orderStatus  order.Status
				expectedFree bool
			}{
				{"Busy Courier 1", order.Assigned, false},
				{"Busy Courier 2", order.Assigned, false},
			},
			expectedFreeCount: 0,
		},
		{
			name: "all completed",
			courierStatusPairs: []struct {
				courierName  string
				orderStatus  order.Status
				expectedFree bool
			}{
				{"Available Courier 1", order.Completed, true},
				{"Available Courier 2", order.Completed, true},
			},
			expectedFreeCount: 2,
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.setupSubtest()
			expectedFreeCouriers := suite.createCouriersAndOrders(ctx, tc.courierStatusPairs)
			suite.verifyFreeCouriers(ctx, tc.expectedFreeCount, expectedFreeCouriers)
		})
	}
}

// setupSubtest prepares a clean environment for each subtest.
func (suite *CourierRepositoryIntegrationTestSuite) setupSubtest() {
	// Clean the database at the start of each subtest to ensure isolation
	suite.Require().NoError(suite.db.Exec("TRUNCATE TABLE storage_places, couriers, orders").Error)

	// Recreate fresh repositories and tracker for each subtest
	suite.tracker = new(MockAggregateTracker)
	suite.courierRepository = courierrepo.NewGormCourierRepository(suite.db, suite.tracker)
	suite.orderRepository = orderrepo.NewGormOrderRepository(suite.db, suite.tracker)
}

// createCouriersAndOrders creates couriers and orders based on test case data.
func (suite *CourierRepositoryIntegrationTestSuite) createCouriersAndOrders(
	ctx context.Context,
	pairs []struct {
		courierName  string
		orderStatus  order.Status
		expectedFree bool
	},
) map[string]bool {
	expectedFreeCouriers := make(map[string]bool)
	for _, pair := range pairs {
		suite.createCourierAndOrder(ctx, pair.courierName, pair.orderStatus)
		if pair.expectedFree {
			expectedFreeCouriers[pair.courierName] = true
		}
	}
	return expectedFreeCouriers
}

// createCourierAndOrder creates a courier and associated order with specified status.
func (suite *CourierRepositoryIntegrationTestSuite) createCourierAndOrder(
	ctx context.Context, courierName string, orderStatus order.Status,
) {
	// Create and add courier
	testCourier := suite.createTestCourierWithName(courierName)
	suite.tracker.On("TrackAggregate", testCourier.ID(), testCourier).Once()
	err := suite.courierRepository.Add(ctx, testCourier)
	suite.Require().NoError(err)

	// Create and add order with appropriate status
	var courierID kernel.UUID
	if orderStatus != order.Created {
		courierID = testCourier.ID()
	}
	testOrder := suite.createTestOrderWithStatus(ctx, courierID, orderStatus)
	suite.tracker.On("TrackAggregate", testOrder.ID(), testOrder).Once()
	err = suite.orderRepository.Add(ctx, testOrder)
	suite.Require().NoError(err)
}

// verifyFreeCouriers checks that the expected couriers are returned as free.
func (suite *CourierRepositoryIntegrationTestSuite) verifyFreeCouriers(
	ctx context.Context, expectedCount int, expectedFreeCouriers map[string]bool,
) {
	// Get all free couriers
	freeCouriers, err := suite.courierRepository.GetAllFree(ctx)
	suite.Require().NoError(err)

	// Verify count
	suite.Len(freeCouriers, expectedCount)

	// Verify correct couriers are returned
	actualFreeCouriers := make(map[string]bool)
	for _, freeCourier := range freeCouriers {
		actualFreeCouriers[freeCourier.Name()] = true
	}

	suite.assertExpectedCouriersAreFree(expectedFreeCouriers, actualFreeCouriers)
	suite.assertNoUnexpectedFreeCouriers(expectedFreeCouriers, actualFreeCouriers)
	suite.tracker.AssertExpectations(suite.T())
}

// assertExpectedCouriersAreFree verifies that all expected free couriers are in the actual results.
func (suite *CourierRepositoryIntegrationTestSuite) assertExpectedCouriersAreFree(
	expected, actual map[string]bool,
) {
	for courierName, shouldBeFree := range expected {
		if shouldBeFree {
			suite.True(actual[courierName], "Courier %s should be free", courierName)
		}
	}
}

// assertNoUnexpectedFreeCouriers verifies that no unexpected couriers are returned as free.
func (suite *CourierRepositoryIntegrationTestSuite) assertNoUnexpectedFreeCouriers(
	expected, actual map[string]bool,
) {
	for courierName := range actual {
		suite.True(expected[courierName], "Unexpected free courier: %s", courierName)
	}
}

// createTestCourier creates a test courier with default values.
func (suite *CourierRepositoryIntegrationTestSuite) createTestCourier() *courier.Courier {
	return suite.createTestCourierWithName("Test Courier")
}

// createTestCourierWithName creates a test courier with specified name.
func (suite *CourierRepositoryIntegrationTestSuite) createTestCourierWithName(name string) *courier.Courier {
	id := kernel.NewUUID()
	location, err := kernel.NewLocation(3, 7)
	suite.Require().NoError(err)

	storagePlaces, err := suite.createTestStoragePlaces()
	suite.Require().NoError(err)

	testCourier, err := courier.NewCourier(id, name, 5, location)
	suite.Require().NoError(err)

	// Add additional storage places if needed (NewCourier creates one by default)
	if len(storagePlaces) > 1 {
		for _, sp := range storagePlaces[1:] {
			err = testCourier.AddStoragePlace(sp.Name(), sp.TotalVolume())
			suite.Require().NoError(err)
		}
	}

	return testCourier
}

// createTestStoragePlaces creates test storage places for courier.
func (suite *CourierRepositoryIntegrationTestSuite) createTestStoragePlaces() ([]*courier.StoragePlace, error) {
	sp1ID := kernel.NewUUID()
	sp1, err := courier.NewStoragePlace(sp1ID, "Bag", 100)
	if err != nil {
		return nil, err
	}

	sp2ID := kernel.NewUUID()
	sp2, err := courier.NewStoragePlace(sp2ID, "Backpack", 150)
	if err != nil {
		return nil, err
	}

	return []*courier.StoragePlace{sp1, sp2}, nil
}

// createTestOrderAssignedToCourier creates a test order assigned to specified courier.
func (suite *CourierRepositoryIntegrationTestSuite) createTestOrderAssignedToCourier(
	_ context.Context, courierID kernel.UUID,
) *order.Order {
	return suite.createTestOrderWithStatus(context.Background(), courierID, order.Assigned)
}

// createTestOrderWithStatus creates a test order with specified status and optional courier assignment.
// For Created status, courierID should be an empty UUID.
// For Assigned and Completed status, courierID should be a valid courier ID.
func (suite *CourierRepositoryIntegrationTestSuite) createTestOrderWithStatus(
	_ context.Context, courierID kernel.UUID, status order.Status,
) *order.Order {
	orderID := kernel.NewUUID()
	location, err := kernel.NewLocation(6, 8)
	suite.Require().NoError(err)

	var courierPtr *kernel.UUID
	if status != order.Created && courierID.Validate() == nil {
		courierPtr = &courierID
	}

	// Use RestoreOrder to create order with desired status
	restoredOrder, err := order.RestoreOrder(orderID, location, 50, status, courierPtr)
	suite.Require().NoError(err)

	return restoredOrder
}

// TestCourierRepository_ErrorScenarios verifies error handling for various failure cases.
func (suite *CourierRepositoryIntegrationTestSuite) TestCourierRepository_ErrorScenarios() {
	testCases := []struct {
		name      string
		operation func() error
		expected  string
	}{
		{
			name: "get with invalid UUID",
			operation: func() error {
				invalidID := kernel.UUID{}
				_, err := suite.courierRepository.Get(context.Background(), invalidID)
				return err
			},
			expected: "required",
		},
		{
			name: "get non-existent courier",
			operation: func() error {
				nonExistentID := kernel.NewUUID()
				_, err := suite.courierRepository.Get(context.Background(), nonExistentID)
				return err
			},
			expected: "not found",
		},
		{
			name: "update non-existent courier",
			operation: func() error {
				// Create courier with minimal storage places to avoid FK issues
				id := kernel.NewUUID()
				location, _ := kernel.NewLocation(3, 7)
				nonExistentCourier, _ := courier.NewCourier(id, "Non-existent", 5, location)
				return suite.courierRepository.Update(context.Background(), nonExistentCourier)
			},
			expected: "foreign key",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.operation()
			suite.Require().Error(err)
			suite.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.expected))
			suite.tracker.AssertExpectations(suite.T())
		})
	}
}

// TestCourierRepository_CapacityManagement verifies business logic around courier capacity.
func (suite *CourierRepositoryIntegrationTestSuite) TestCourierRepository_CapacityManagement() {
	ctx := context.Background()

	// Create courier with specific storage capacity
	testCourier := suite.createTestCourier()
	suite.tracker.On("TrackAggregate", testCourier.ID(), testCourier).Once()
	err := suite.courierRepository.Add(ctx, testCourier)
	suite.Require().NoError(err)

	// Try to assign orders one by one until capacity is reached
	successfulAssignments := 0
	for range len(testCourier.StoragePlaces()) + 1 {
		// Create a small order that fits in storage
		orderID := kernel.NewUUID()
		location, locationErr := kernel.NewLocation(6, 8)
		suite.Require().NoError(locationErr)
		testOrder, orderErr := order.NewOrder(orderID, location, 5) // Small volume
		suite.Require().NoError(orderErr)

		// Try to take the order
		canTake, takeErr := testCourier.CanTakeOrder(testOrder)
		suite.Require().NoError(takeErr)
		if canTake {
			err = testCourier.TakeOrder(testOrder)
			suite.Require().NoError(err)
			successfulAssignments++
		} else {
			break // Capacity reached
		}
	}

	// Update courier with assignments
	suite.tracker.On("TrackAggregate", testCourier.ID(), testCourier).Once()
	err = suite.courierRepository.Update(ctx, testCourier)
	suite.Require().NoError(err)

	// Verify capacity constraints were respected
	suite.LessOrEqual(successfulAssignments, len(testCourier.StoragePlaces()),
		"Should not assign more orders than storage capacity")
	suite.Positive(successfulAssignments, "Should assign at least one order")

	// Retrieve and verify persisted state
	retrievedCourier, err := suite.courierRepository.Get(ctx, testCourier.ID())
	suite.Require().NoError(err)

	occupiedPlaces := 0
	for _, place := range retrievedCourier.StoragePlaces() {
		if place.OrderID() != nil {
			occupiedPlaces++
		}
	}
	suite.Equal(successfulAssignments, occupiedPlaces,
		"Persisted state should match assigned orders")

	suite.tracker.AssertExpectations(suite.T())
}

// assertCourierCount verifies the number of couriers in the database.
func (suite *CourierRepositoryIntegrationTestSuite) assertCourierCount(expected int) {
	var count int64
	err := suite.db.Model(&courierrepo.CourierDTO{}).Count(&count).Error
	suite.Require().NoError(err)
	suite.Equal(int64(expected), count)
}

// assertStoragePlaceCount verifies the number of storage places in the database.
func (suite *CourierRepositoryIntegrationTestSuite) assertStoragePlaceCount(expected int) {
	var count int64
	err := suite.db.Model(&courierrepo.StoragePlaceDTO{}).Count(&count).Error
	suite.Require().NoError(err)
	suite.Equal(int64(expected), count)
}

func TestCourierRepositoryIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CourierRepositoryIntegrationTestSuite))
}
