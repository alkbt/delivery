package orderrepo_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"delivery/internal/adapters/out/postgres/orderrepo"
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

// OrderRepositoryIntegrationTestSuite provides integration tests for OrderRepository
// using PostgreSQL containers to verify database persistence behavior.
type OrderRepositoryIntegrationTestSuite struct {
	suite.Suite
	container  *postgres.PostgresContainer
	db         *gorm.DB
	repository *orderrepo.GormOrderRepository
	tracker    *MockAggregateTracker
}

func (suite *OrderRepositoryIntegrationTestSuite) SetupSuite() {
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
	suite.Require().NoError(db.AutoMigrate(&orderrepo.OrderDTO{}))
}

func (suite *OrderRepositoryIntegrationTestSuite) SetupTest() {
	// Clean the database before each test
	suite.Require().NoError(suite.db.Exec("TRUNCATE TABLE orders").Error)

	// Create fresh repository and tracker for each test
	suite.tracker = new(MockAggregateTracker)
	suite.repository = orderrepo.NewGormOrderRepository(suite.db, suite.tracker)
}

func (suite *OrderRepositoryIntegrationTestSuite) TearDownSuite() {
	if suite.container != nil {
		suite.Require().NoError(suite.container.Terminate(context.Background()))
	}
}

func (suite *OrderRepositoryIntegrationTestSuite) TestAdd_ValidOrder_Success() {
	ctx := context.Background()

	// Create valid order
	testOrder := suite.createTestOrder()

	// Set expectations on mock
	suite.tracker.On("TrackAggregate", testOrder.ID(), testOrder).Once()

	// Add order to repository
	err := suite.repository.Add(ctx, testOrder)
	suite.Require().NoError(err)

	// Verify order was persisted
	suite.assertOrderCount(1)

	// Assert that all expectations were met
	suite.tracker.AssertExpectations(suite.T())
}

func (suite *OrderRepositoryIntegrationTestSuite) TestAdd_InvalidOrder_BusinessRules() {
	testCases := []struct {
		name     string
		setup    func() (*order.Order, error)
		expected string
	}{
		{
			name: "invalid volume",
			setup: func() (*order.Order, error) {
				id := kernel.NewUUID()
				location, _ := kernel.NewLocation(5, 5)
				return order.RestoreOrder(id, location, -1, order.Created, nil)
			},
			expected: "volume",
		},
		{
			name: "invalid location coordinates",
			setup: func() (*order.Order, error) {
				id := kernel.NewUUID()
				// NewLocation will fail with invalid coordinates
				location, err := kernel.NewLocation(-1, 5)
				if err != nil {
					return nil, err
				}
				return order.RestoreOrder(id, location, 50, order.Created, nil)
			},
			expected: "min value",
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create invalid order
			invalidOrder, err := tc.setup()
			if err != nil {
				// Constructor validation failed as expected
				suite.Contains(err.Error(), tc.expected)
				return
			}

			// Try to add invalid order
			err = suite.repository.Add(ctx, invalidOrder)
			suite.Require().Error(err)
			suite.Contains(err.Error(), tc.expected)

			// Verify no order was persisted
			suite.assertOrderCount(0)
			suite.tracker.AssertExpectations(suite.T())
		})
	}
}

func (suite *OrderRepositoryIntegrationTestSuite) TestGet_ExistingOrder_ReturnsOrder() {
	ctx := context.Background()

	// Create and add order
	id := kernel.NewUUID()
	location, err := kernel.NewLocation(5, 8)
	suite.Require().NoError(err)

	originalOrder, err := order.NewOrder(id, location, 75)
	suite.Require().NoError(err)

	// Set expectations for Add operation
	suite.tracker.On("TrackAggregate", id, originalOrder).Once()

	err = suite.repository.Add(ctx, originalOrder)
	suite.Require().NoError(err)

	// Retrieve order
	retrievedOrder, err := suite.repository.Get(ctx, id)
	suite.Require().NoError(err)

	// Verify order details
	suite.Equal(id, retrievedOrder.ID())
	suite.Equal(location.X(), retrievedOrder.Location().X())
	suite.Equal(location.Y(), retrievedOrder.Location().Y())
	suite.Equal(75, retrievedOrder.Volume())
	suite.Equal(order.Created, retrievedOrder.Status())
	suite.Nil(retrievedOrder.Courier())

	// Assert that all expectations were met
	suite.tracker.AssertExpectations(suite.T())
}

func (suite *OrderRepositoryIntegrationTestSuite) TestGet_NonExistentOrder_ReturnsNotFoundError() {
	ctx := context.Background()

	// Try to get non-existent order
	nonExistentID := kernel.NewUUID()
	retrievedOrder, err := suite.repository.Get(ctx, nonExistentID)

	// Verify error and result
	suite.Nil(retrievedOrder)
	suite.Require().Error(err)

	var notFoundErr *errs.ObjectNotFoundError
	suite.Require().ErrorAs(err, &notFoundErr)

	// Assert no unexpected calls
	suite.tracker.AssertExpectations(suite.T())
}

func (suite *OrderRepositoryIntegrationTestSuite) TestUpdate_OrderStatusTransitions() {
	testCases := []struct {
		name          string
		initialStatus order.Status
		updatedStatus order.Status
		setupCourier  bool
		verify        func(*order.Order)
	}{
		{
			name:          "created to assigned",
			initialStatus: order.Created,
			updatedStatus: order.Assigned,
			setupCourier:  true,
			verify: func(o *order.Order) {
				suite.Equal(order.Assigned, o.Status())
				suite.NotNil(o.Courier())
			},
		},
		{
			name:          "assigned to completed",
			initialStatus: order.Assigned,
			updatedStatus: order.Completed,
			setupCourier:  true,
			verify: func(o *order.Order) {
				suite.Equal(order.Completed, o.Status())
				suite.NotNil(o.Courier())
			},
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create initial order
			var courierID *kernel.UUID
			if tc.initialStatus != order.Created {
				cid := kernel.NewUUID()
				courierID = &cid
			}

			initialOrder := suite.createTestOrderWithStatus(tc.initialStatus, courierID)
			suite.tracker.On("TrackAggregate", initialOrder.ID(), initialOrder).Once()
			err := suite.repository.Add(ctx, initialOrder)
			suite.Require().NoError(err)

			// Update order status
			var updatedCourierID *kernel.UUID
			if tc.setupCourier {
				if courierID != nil {
					updatedCourierID = courierID
				} else {
					cid := kernel.NewUUID()
					updatedCourierID = &cid
				}
			}

			updatedOrder, err := order.RestoreOrder(
				initialOrder.ID(),
				initialOrder.Location(),
				initialOrder.Volume(),
				tc.updatedStatus,
				updatedCourierID,
			)
			suite.Require().NoError(err)

			suite.tracker.On("TrackAggregate", updatedOrder.ID(), updatedOrder).Once()
			err = suite.repository.Update(ctx, updatedOrder)
			suite.Require().NoError(err)

			// Retrieve and verify updated order
			retrievedOrder, err := suite.repository.Get(ctx, initialOrder.ID())
			suite.Require().NoError(err)
			tc.verify(retrievedOrder)

			suite.tracker.AssertExpectations(suite.T())
		})
	}
}

func (suite *OrderRepositoryIntegrationTestSuite) TestUpdate_NonExistentOrder_ReturnsError() {
	ctx := context.Background()

	// Create order that doesn't exist in database
	id := kernel.NewUUID()
	location, err := kernel.NewLocation(5, 5)
	suite.Require().NoError(err)

	nonExistentOrder, err := order.NewOrder(id, location, 50)
	suite.Require().NoError(err)

	// No expectations on tracker since operation should fail

	// Try to update non-existent order
	err = suite.repository.Update(ctx, nonExistentOrder)
	suite.Require().Error(err)

	// Assert no unexpected calls
	suite.tracker.AssertExpectations(suite.T())
}

func (suite *OrderRepositoryIntegrationTestSuite) TestGetFirstInCreatedStatus_OrdersExist_ReturnsFirstCreatedOrder() {
	ctx := context.Background()

	// Setup mock expectations for mixed statuses
	suite.setupMockExpectationsForMixedStatuses()

	// Create multiple orders with different statuses
	orders := suite.createTestOrdersWithDifferentStatuses(ctx)

	// Get first order in Created status
	retrievedOrder, err := suite.repository.GetFirstInCreatedStatus(ctx)
	suite.Require().NoError(err)

	// Verify it's one of the created orders
	suite.Equal(order.Created, retrievedOrder.Status())

	// Verify it's one of our created orders
	found := false
	for _, testOrder := range orders {
		if testOrder.Status() == order.Created && testOrder.ID() == retrievedOrder.ID() {
			found = true
			break
		}
	}
	suite.True(found, "Retrieved order should be one of the test orders in Created status")

	// Assert all mock expectations were met
	suite.tracker.AssertExpectations(suite.T())
}

func (suite *OrderRepositoryIntegrationTestSuite) TestGetFirstInCreatedStatus_NoCreatedOrders_ReturnsNotFoundError() {
	ctx := context.Background()

	// Setup mock expectations for the orders we'll create
	suite.setupMockExpectationsForNonCreatedOrders()

	// Create only assigned/completed orders
	suite.createOrderWithStatus(ctx, order.Assigned)
	suite.createOrderWithStatus(ctx, order.Completed)

	// Try to get first created order
	retrievedOrder, err := suite.repository.GetFirstInCreatedStatus(ctx)

	// Verify error and result
	suite.Nil(retrievedOrder)
	suite.Require().Error(err)

	var notFoundErr *errs.ObjectNotFoundError
	suite.Require().ErrorAs(err, &notFoundErr)

	// Assert all expectations were met
	suite.tracker.AssertExpectations(suite.T())
}

func (suite *OrderRepositoryIntegrationTestSuite) TestGetAllInAssignedStatus_AssignedOrdersExist_ReturnsAllAssignedOrders() {
	ctx := context.Background()

	// Create orders with different statuses
	suite.setupMockExpectationsForMixedStatuses()
	suite.createTestOrdersWithDifferentStatuses(ctx)

	// Get all assigned orders
	assignedOrders, err := suite.repository.GetAllInAssignedStatus(ctx)
	suite.Require().NoError(err)

	// Verify all returned orders have Assigned status
	for _, assignedOrder := range assignedOrders {
		suite.Equal(order.Assigned, assignedOrder.Status())
		suite.NotNil(assignedOrder.Courier(), "Assigned orders should have courier assigned")
	}

	// Verify we got the correct number of assigned orders
	suite.GreaterOrEqual(len(assignedOrders), 1)

	// Assert all expectations were met
	suite.tracker.AssertExpectations(suite.T())
}

func (suite *OrderRepositoryIntegrationTestSuite) TestGetAllInAssignedStatus_NoAssignedOrders_ReturnsEmptySlice() {
	ctx := context.Background()

	// Setup mock expectations for non-assigned orders
	suite.setupMockExpectationsForNonAssignedOrders()

	// Create only created/completed orders
	suite.createOrderWithStatus(ctx, order.Created)
	suite.createOrderWithStatus(ctx, order.Completed)

	// Get all assigned orders
	assignedOrders, err := suite.repository.GetAllInAssignedStatus(ctx)
	suite.Require().NoError(err)

	// Verify empty result
	suite.Empty(assignedOrders)

	// Assert all expectations were met
	suite.tracker.AssertExpectations(suite.T())
}

// setupMockExpectationsForMixedStatuses sets up mock expectations for orders with different statuses.
func (suite *OrderRepositoryIntegrationTestSuite) setupMockExpectationsForMixedStatuses() {
	// Expect multiple TrackAggregate calls for different orders
	suite.tracker.On("TrackAggregate", mock.AnythingOfType("kernel.UUID"), mock.Anything).Times(5)
}

// setupMockExpectationsForNonCreatedOrders sets up expectations for orders that are not in Created status.
func (suite *OrderRepositoryIntegrationTestSuite) setupMockExpectationsForNonCreatedOrders() {
	suite.tracker.On("TrackAggregate", mock.AnythingOfType("kernel.UUID"), mock.Anything).Times(2)
}

// setupMockExpectationsForNonAssignedOrders sets up expectations for orders that are not in Assigned status.
func (suite *OrderRepositoryIntegrationTestSuite) setupMockExpectationsForNonAssignedOrders() {
	suite.tracker.On("TrackAggregate", mock.AnythingOfType("kernel.UUID"), mock.Anything).Times(2)
}

// createTestOrdersWithDifferentStatuses creates orders with various statuses for testing.
func (suite *OrderRepositoryIntegrationTestSuite) createTestOrdersWithDifferentStatuses(
	ctx context.Context,
) []*order.Order {
	var orders []*order.Order

	// Create orders in different statuses
	statuses := []order.Status{order.Created, order.Created, order.Assigned, order.Assigned, order.Completed}

	for i, status := range statuses {
		location, err := kernel.NewLocation(kernel.Coordinate(1+i%10), kernel.Coordinate(1+(i*2)%10))
		suite.Require().NoError(err)

		id := kernel.NewUUID()
		domainOrder, err := order.RestoreOrder(id, location, 50+i*5, status, suite.getCourierForStatus(status))
		suite.Require().NoError(err)

		err = suite.repository.Add(ctx, domainOrder)
		suite.Require().NoError(err)

		orders = append(orders, domainOrder)
	}

	return orders
}

// createOrderWithStatus creates an order with specified status.
func (suite *OrderRepositoryIntegrationTestSuite) createOrderWithStatus(
	ctx context.Context, status order.Status,
) *order.Order {
	id := kernel.NewUUID()
	location, err := kernel.NewLocation(5, 5)
	suite.Require().NoError(err)

	var courierID *kernel.UUID
	if status == order.Assigned || status == order.Completed {
		cid := kernel.NewUUID()
		courierID = &cid
	}

	testOrder, err := order.RestoreOrder(id, location, 50, status, courierID)
	suite.Require().NoError(err)

	err = suite.repository.Add(ctx, testOrder)
	suite.Require().NoError(err)

	return testOrder
}

// getCourierForStatus returns a courier ID for statuses that require one.
func (suite *OrderRepositoryIntegrationTestSuite) getCourierForStatus(status order.Status) *kernel.UUID {
	if status == order.Assigned || status == order.Completed {
		cid := kernel.NewUUID()
		return &cid
	}
	return nil
}

// TestOrderRepository_ErrorScenarios verifies error handling for various failure cases.
func (suite *OrderRepositoryIntegrationTestSuite) TestOrderRepository_ErrorScenarios() {
	testCases := []struct {
		name      string
		operation func() error
		expected  string
	}{
		{
			name: "get with invalid UUID",
			operation: func() error {
				invalidID := kernel.UUID{}
				_, err := suite.repository.Get(context.Background(), invalidID)
				return err
			},
			expected: "required",
		},
		{
			name: "get non-existent order",
			operation: func() error {
				nonExistentID := kernel.NewUUID()
				_, err := suite.repository.Get(context.Background(), nonExistentID)
				return err
			},
			expected: "not found",
		},
		{
			name: "update non-existent order",
			operation: func() error {
				nonExistentOrder := suite.createTestOrder()
				return suite.repository.Update(context.Background(), nonExistentOrder)
			},
			expected: "record not found",
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

// TestOrderRepository_QueryLogic verifies business logic in query methods.
func (suite *OrderRepositoryIntegrationTestSuite) TestOrderRepository_QueryLogic() {
	ctx := context.Background()

	// Create orders in different statuses
	createdOrder1 := suite.createTestOrderWithStatus(order.Created, nil)
	createdOrder2 := suite.createTestOrderWithStatus(order.Created, nil)
	courierID := kernel.NewUUID()
	assignedOrder := suite.createTestOrderWithStatus(order.Assigned, &courierID)
	completedOrder := suite.createTestOrderWithStatus(order.Completed, &courierID)

	// Set up expectations
	suite.tracker.On("TrackAggregate", createdOrder1.ID(), createdOrder1).Once()
	suite.tracker.On("TrackAggregate", createdOrder2.ID(), createdOrder2).Once()
	suite.tracker.On("TrackAggregate", assignedOrder.ID(), assignedOrder).Once()
	suite.tracker.On("TrackAggregate", completedOrder.ID(), completedOrder).Once()

	// Add all orders
	err := suite.repository.Add(ctx, createdOrder1)
	suite.Require().NoError(err)
	err = suite.repository.Add(ctx, createdOrder2)
	suite.Require().NoError(err)
	err = suite.repository.Add(ctx, assignedOrder)
	suite.Require().NoError(err)
	err = suite.repository.Add(ctx, completedOrder)
	suite.Require().NoError(err)

	// Test GetFirstInCreatedStatus
	firstCreated, err := suite.repository.GetFirstInCreatedStatus(ctx)
	suite.Require().NoError(err)
	suite.Equal(order.Created, firstCreated.Status())
	suite.Nil(firstCreated.Courier())

	// Test GetAllInAssignedStatus
	allAssigned, err := suite.repository.GetAllInAssignedStatus(ctx)
	suite.Require().NoError(err)
	suite.Len(allAssigned, 1)
	suite.Equal(order.Assigned, allAssigned[0].Status())
	suite.NotNil(allAssigned[0].Courier())
	suite.Equal(courierID, *allAssigned[0].Courier())

	suite.tracker.AssertExpectations(suite.T())
}

// TestOrderRepository_Concurrency verifies repository behavior under concurrent access.
func (suite *OrderRepositoryIntegrationTestSuite) TestOrderRepository_Concurrency() {
	ctx := context.Background()

	// Create initial order
	initialOrder := suite.createTestOrder()
	suite.tracker.On("TrackAggregate", initialOrder.ID(), initialOrder).Once()
	err := suite.repository.Add(ctx, initialOrder)
	suite.Require().NoError(err)

	// Simulate concurrent reads
	results := make(chan *order.Order, 3)
	errors := make(chan error, 3)

	for range 3 {
		go func() {
			retrievedOrder, readErr := suite.repository.Get(ctx, initialOrder.ID())
			if readErr != nil {
				errors <- readErr
			} else {
				results <- retrievedOrder
			}
		}()
	}

	// Collect results
	for range 3 {
		select {
		case result := <-results:
			suite.Equal(initialOrder.ID(), result.ID())
		case readErr := <-errors:
			suite.Failf("Unexpected error in concurrent read", "%v", readErr)
		}
	}

	suite.tracker.AssertExpectations(suite.T())
}

// createTestOrder creates a basic test order with default values.
func (suite *OrderRepositoryIntegrationTestSuite) createTestOrder() *order.Order {
	id := kernel.NewUUID()
	location, err := kernel.NewLocation(5, 5)
	suite.Require().NoError(err)
	testOrder, err := order.NewOrder(id, location, 50)
	suite.Require().NoError(err)
	return testOrder
}

// createTestOrderWithStatus creates a test order with specified status and optional courier.
func (suite *OrderRepositoryIntegrationTestSuite) createTestOrderWithStatus(
	status order.Status, courierID *kernel.UUID,
) *order.Order {
	id := kernel.NewUUID()
	location, err := kernel.NewLocation(5, 5)
	suite.Require().NoError(err)
	testOrder, err := order.RestoreOrder(id, location, 50, status, courierID)
	suite.Require().NoError(err)
	return testOrder
}

// assertOrderCount verifies the number of orders in the database.
func (suite *OrderRepositoryIntegrationTestSuite) assertOrderCount(expected int) {
	var count int64
	err := suite.db.Model(&orderrepo.OrderDTO{}).Count(&count).Error
	suite.Require().NoError(err)
	suite.Equal(int64(expected), count)
}

func TestOrderRepositoryIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(OrderRepositoryIntegrationTestSuite))
}
