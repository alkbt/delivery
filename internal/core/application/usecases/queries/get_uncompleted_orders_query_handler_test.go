package queries_test

import (
	"context"
	"testing"
	"time"

	"delivery/internal/adapters/out/postgres/courierrepo"
	"delivery/internal/adapters/out/postgres/orderrepo"
	"delivery/internal/core/application/usecases/queries"
	"delivery/internal/core/domain/model/courier"
	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/core/domain/model/order"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gorm_postgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type GetUncompletedOrdersQueryHandlerTestSuite struct {
	suite.Suite
	container   *postgres.PostgresContainer
	db          *gorm.DB
	handler     queries.GetUncompletedOrdersQueryHandler
	orderRepo   *orderrepo.GormOrderRepository
	courierRepo *courierrepo.GormCourierRepository
	testCourier *courier.Courier
}

func (suite *GetUncompletedOrdersQueryHandlerTestSuite) SetupSuite() {
	ctx := context.Background()

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

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	suite.Require().NoError(err)

	db, err := gorm.Open(gorm_postgres.Open(dsn), &gorm.Config{})
	suite.Require().NoError(err)
	suite.db = db

	err = db.AutoMigrate(&orderrepo.OrderDTO{}, &courierrepo.CourierDTO{}, &courierrepo.StoragePlaceDTO{})
	suite.Require().NoError(err)

	suite.handler = queries.NewGetUncompletedOrdersQueryHandler(db)
	suite.orderRepo = orderrepo.NewGormOrderRepository(db, &mockAggregateTracker{})
	suite.courierRepo = courierrepo.NewGormCourierRepository(db, &mockAggregateTracker{})

	// Create a test courier for assigned orders
	location, err := kernel.NewLocation(5, 5)
	suite.Require().NoError(err)
	suite.testCourier, err = courier.NewCourier(kernel.NewUUID(), "Test Courier", 3, location)
	suite.Require().NoError(err)
	err = suite.courierRepo.Add(ctx, suite.testCourier)
	suite.Require().NoError(err)
}

func (suite *GetUncompletedOrdersQueryHandlerTestSuite) TearDownSuite() {
	if suite.container != nil {
		err := suite.container.Terminate(context.Background())
		suite.Require().NoError(err)
	}
}

func (suite *GetUncompletedOrdersQueryHandlerTestSuite) SetupTest() {
	err := suite.db.Exec("TRUNCATE TABLE orders CASCADE").Error
	suite.Require().NoError(err)
}

func (suite *GetUncompletedOrdersQueryHandlerTestSuite) TestHandle_EmptyDatabase_ReturnsEmptySlice() {
	query := queries.NewGetUncompletedOrdersQuery()

	result, err := suite.handler.Handle(context.Background(), query)

	suite.Require().NoError(err)
	suite.NotNil(result)
	suite.Empty(result)
}

func (suite *GetUncompletedOrdersQueryHandlerTestSuite) TestHandle_WithOnlyCompletedOrders_ReturnsEmptySlice() {
	// Create and complete an order
	location, _ := kernel.NewLocation(3, 4)
	order1, _ := order.NewOrder(kernel.NewUUID(), location, 10)
	err := order1.Assign(suite.testCourier.ID())
	suite.Require().NoError(err)
	err = order1.Complete()
	suite.Require().NoError(err)
	err = suite.orderRepo.Add(context.Background(), order1)
	suite.Require().NoError(err)

	// Create and complete another order
	location2, _ := kernel.NewLocation(7, 8)
	order2, _ := order.NewOrder(kernel.NewUUID(), location2, 15)
	err = order2.Assign(suite.testCourier.ID())
	suite.Require().NoError(err)
	err = order2.Complete()
	suite.Require().NoError(err)
	err = suite.orderRepo.Add(context.Background(), order2)
	suite.Require().NoError(err)

	query := queries.NewGetUncompletedOrdersQuery()

	result, err := suite.handler.Handle(context.Background(), query)

	suite.Require().NoError(err)
	suite.NotNil(result)
	suite.Empty(result)
}

func (suite *GetUncompletedOrdersQueryHandlerTestSuite) TestHandle_WithMixedStatuses_ReturnsOnlyUncompleted() {
	// Create orders with different statuses
	createdOrders := suite.createCreatedOrders()
	assignedOrders := suite.createAssignedOrders()
	completedOrders := suite.createCompletedOrders()

	// Save all orders
	for _, o := range append(append(createdOrders, assignedOrders...), completedOrders...) {
		err := suite.orderRepo.Add(context.Background(), o)
		suite.Require().NoError(err)
	}

	query := queries.NewGetUncompletedOrdersQuery()

	result, err := suite.handler.Handle(context.Background(), query)

	suite.Require().NoError(err)
	suite.Len(result, 4) // 2 created + 2 assigned

	// Verify all results are uncompleted orders
	resultIDs := make(map[kernel.UUID]bool)
	for _, r := range result {
		resultIDs[r.ID] = true
	}

	// Check that all created and assigned orders are in results
	for _, o := range append(createdOrders, assignedOrders...) {
		suite.True(resultIDs[o.ID()], "Order %s should be in results", o.ID())
	}

	// Check that no completed orders are in results
	for _, o := range completedOrders {
		suite.False(resultIDs[o.ID()], "Completed order %s should not be in results", o.ID())
	}
}

func (suite *GetUncompletedOrdersQueryHandlerTestSuite) TestHandle_InvalidQuery_ReturnsError() {
	invalidQuery := queries.GetUncompletedOrdersQuery{}

	result, err := suite.handler.Handle(context.Background(), invalidQuery)

	suite.Require().Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "must be created via NewGetUncompletedOrdersQuery constructor")
}

func (suite *GetUncompletedOrdersQueryHandlerTestSuite) TestHandle_ContextCancellation_ReturnsError() {
	// Create many orders to ensure context cancellation happens during processing
	for range 50 {
		location, _ := kernel.NewRandomLocation()
		o, _ := order.NewOrder(kernel.NewUUID(), location, 10)
		err := suite.orderRepo.Add(context.Background(), o)
		suite.Require().NoError(err)
	}

	query := queries.NewGetUncompletedOrdersQuery()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := suite.handler.Handle(ctx, query)

	suite.Require().Error(err)
	suite.Nil(result)
}

func (suite *GetUncompletedOrdersQueryHandlerTestSuite) TestHandle_VariousLocations_CorrectlyMapsCoordinates() {
	testCases := []struct {
		name string
		x    kernel.Coordinate
		y    kernel.Coordinate
	}{
		{"Order at Origin", 1, 1},
		{"Order at Max", 10, 10},
		{"Order at Center", 5, 5},
		{"Order at Mixed", 3, 8},
	}

	ordersByLocation := make(map[string]*order.Order)
	for _, tc := range testCases {
		location, err := kernel.NewLocation(tc.x, tc.y)
		suite.Require().NoError(err)

		o, err := order.NewOrder(kernel.NewUUID(), location, 10)
		suite.Require().NoError(err)

		err = suite.orderRepo.Add(context.Background(), o)
		suite.Require().NoError(err)

		ordersByLocation[tc.name] = o
	}

	query := queries.NewGetUncompletedOrdersQuery()

	result, err := suite.handler.Handle(context.Background(), query)

	suite.Require().NoError(err)
	suite.Len(result, len(testCases))

	// Map results by ID for verification
	resultMap := make(map[kernel.UUID]queries.GetUncompletedOrdersQueryResponse)
	for _, r := range result {
		resultMap[r.ID] = r
	}

	// Verify each order's location
	for name, o := range ordersByLocation {
		result, exists := resultMap[o.ID()]
		suite.True(exists, "Order %s not found in results", name)

		isEqual, locErr := o.Location().IsEqual(result.Location)
		suite.Require().NoError(locErr)
		suite.True(isEqual, "Location mismatch for order %s", name)
	}
}

func (suite *GetUncompletedOrdersQueryHandlerTestSuite) TestHandle_OrdersAreSortedByID() {
	// Create orders with specific IDs to verify sorting
	orders := make([]*order.Order, 3)
	locations := []kernel.Location{}

	for i := range 3 {
		loc, _ := kernel.NewLocation(kernel.Coordinate(i+1), kernel.Coordinate(i+1))
		locations = append(locations, loc)
	}

	// Create orders in reverse order to test sorting
	for i := 2; i >= 0; i-- {
		o, err := order.NewOrder(kernel.NewUUID(), locations[i], 10)
		suite.Require().NoError(err)
		orders[i] = o
		err = suite.orderRepo.Add(context.Background(), o)
		suite.Require().NoError(err)
	}

	query := queries.NewGetUncompletedOrdersQuery()

	result, err := suite.handler.Handle(context.Background(), query)

	suite.Require().NoError(err)
	suite.Len(result, 3)

	// Results should be sorted by ID
	for i := range len(result) - 1 {
		suite.Less(result[i].ID.String(), result[i+1].ID.String(),
			"Orders should be sorted by ID: %s should come before %s",
			result[i].ID, result[i+1].ID)
	}
}

func (suite *GetUncompletedOrdersQueryHandlerTestSuite) createCreatedOrders() []*order.Order {
	orders := make([]*order.Order, 0)

	location1, _ := kernel.NewLocation(2, 3)
	order1, _ := order.NewOrder(kernel.NewUUID(), location1, 5)
	orders = append(orders, order1)

	location2, _ := kernel.NewLocation(8, 9)
	order2, _ := order.NewOrder(kernel.NewUUID(), location2, 12)
	orders = append(orders, order2)

	return orders
}

func (suite *GetUncompletedOrdersQueryHandlerTestSuite) createAssignedOrders() []*order.Order {
	orders := make([]*order.Order, 0)

	location1, _ := kernel.NewLocation(4, 5)
	order1, _ := order.NewOrder(kernel.NewUUID(), location1, 8)
	_ = order1.Assign(suite.testCourier.ID())
	orders = append(orders, order1)

	location2, _ := kernel.NewLocation(6, 7)
	order2, _ := order.NewOrder(kernel.NewUUID(), location2, 15)
	_ = order2.Assign(suite.testCourier.ID())
	orders = append(orders, order2)

	return orders
}

func (suite *GetUncompletedOrdersQueryHandlerTestSuite) createCompletedOrders() []*order.Order {
	orders := make([]*order.Order, 0)

	location1, _ := kernel.NewLocation(1, 2)
	order1, _ := order.NewOrder(kernel.NewUUID(), location1, 20)
	_ = order1.Assign(suite.testCourier.ID())
	_ = order1.Complete()
	orders = append(orders, order1)

	location2, _ := kernel.NewLocation(9, 10)
	order2, _ := order.NewOrder(kernel.NewUUID(), location2, 25)
	_ = order2.Assign(suite.testCourier.ID())
	_ = order2.Complete()
	orders = append(orders, order2)

	return orders
}

func TestGetUncompletedOrdersQueryHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GetUncompletedOrdersQueryHandlerTestSuite))
}
