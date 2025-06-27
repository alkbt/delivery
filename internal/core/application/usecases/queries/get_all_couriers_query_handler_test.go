package queries_test

import (
	"context"
	"testing"
	"time"

	"delivery/internal/adapters/out/postgres/courierrepo"
	"delivery/internal/core/application/usecases/queries"
	"delivery/internal/core/domain/model/courier"
	"delivery/internal/core/domain/model/kernel"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gorm_postgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type GetAllCouriersQueryHandlerTestSuite struct {
	suite.Suite
	container *postgres.PostgresContainer
	db        *gorm.DB
	handler   queries.GetAllCouriersQueryHandler
}

func (suite *GetAllCouriersQueryHandlerTestSuite) SetupSuite() {
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

	err = db.AutoMigrate(&courierrepo.CourierDTO{}, &courierrepo.StoragePlaceDTO{})
	suite.Require().NoError(err)

	suite.handler = queries.NewGetAllCouriersQueryHandler(db)
}

func (suite *GetAllCouriersQueryHandlerTestSuite) TearDownSuite() {
	if suite.container != nil {
		err := suite.container.Terminate(context.Background())
		suite.Require().NoError(err)
	}
}

func (suite *GetAllCouriersQueryHandlerTestSuite) SetupTest() {
	err := suite.db.Exec("TRUNCATE TABLE couriers CASCADE").Error
	suite.Require().NoError(err)
}

func (suite *GetAllCouriersQueryHandlerTestSuite) TestHandle_EmptyDatabase_ReturnsEmptySlice() {
	query := queries.NewGetAllCouriersQuery()

	result, err := suite.handler.Handle(context.Background(), query)

	suite.Require().NoError(err)
	suite.NotNil(result)
	suite.Empty(result)
}

func (suite *GetAllCouriersQueryHandlerTestSuite) TestHandle_WithCouriers_ReturnsAllCouriersOrderedByName() {
	couriers := suite.createTestCouriers()
	suite.saveCouriers(couriers)

	query := queries.NewGetAllCouriersQuery()

	result, err := suite.handler.Handle(context.Background(), query)

	suite.Require().NoError(err)
	suite.Len(result, 3)

	suite.Equal("Alice", result[0].Name)
	suite.Equal(couriers[0].ID(), result[0].ID)
	isEqual, err := couriers[0].Location().IsEqual(result[0].Location)
	suite.Require().NoError(err)
	suite.True(isEqual)

	suite.Equal("Bob", result[1].Name)
	suite.Equal(couriers[1].ID(), result[1].ID)
	isEqual, err = couriers[1].Location().IsEqual(result[1].Location)
	suite.Require().NoError(err)
	suite.True(isEqual)

	suite.Equal("Charlie", result[2].Name)
	suite.Equal(couriers[2].ID(), result[2].ID)
	isEqual, err = couriers[2].Location().IsEqual(result[2].Location)
	suite.Require().NoError(err)
	suite.True(isEqual)
}

func (suite *GetAllCouriersQueryHandlerTestSuite) TestHandle_InvalidQuery_ReturnsError() {
	invalidQuery := queries.GetAllCouriersQuery{}

	result, err := suite.handler.Handle(context.Background(), invalidQuery)

	suite.Require().Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "must be created via NewGetAllCouriersQuery constructor")
}

func (suite *GetAllCouriersQueryHandlerTestSuite) TestHandle_ContextCancellation_ReturnsError() {
	suite.createAndSaveLargeCourierSet()

	query := queries.NewGetAllCouriersQuery()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := suite.handler.Handle(ctx, query)

	suite.Require().Error(err)
	suite.Nil(result)
}

func (suite *GetAllCouriersQueryHandlerTestSuite) TestHandle_VariousLocations_CorrectlyMapsCoordinates() {
	testCases := []struct {
		name string
		x    kernel.Coordinate
		y    kernel.Coordinate
	}{
		{"Courier at Origin", 1, 1},
		{"Courier at Max", 10, 10},
		{"Courier at Center", 5, 5},
		{"Courier at Mixed", 3, 8},
	}

	for _, tc := range testCases {
		location, err := kernel.NewLocation(tc.x, tc.y)
		suite.Require().NoError(err)

		courier, err := courier.NewCourier(
			kernel.NewUUID(),
			tc.name,
			3,
			location,
		)
		suite.Require().NoError(err)

		repo := courierrepo.NewGormCourierRepository(suite.db, &mockAggregateTracker{})
		err = repo.Add(context.Background(), courier)
		suite.Require().NoError(err)
	}

	query := queries.NewGetAllCouriersQuery()

	result, err := suite.handler.Handle(context.Background(), query)

	suite.Require().NoError(err)
	suite.Len(result, len(testCases))

	resultMap := make(map[string]queries.GetAllCouriersQueryResponse)
	for _, r := range result {
		resultMap[r.Name] = r
	}

	for _, tc := range testCases {
		courier, exists := resultMap[tc.name]
		suite.True(exists, "Courier %s not found", tc.name)
		suite.Equal(tc.x, courier.Location.X())
		suite.Equal(tc.y, courier.Location.Y())
	}
}

func (suite *GetAllCouriersQueryHandlerTestSuite) createTestCouriers() []*courier.Courier {
	couriers := make([]*courier.Courier, 0)

	location1, _ := kernel.NewLocation(3, 4)
	courier1, _ := courier.NewCourier(kernel.NewUUID(), "Alice", 3, location1)
	courier1.AddStoragePlace("Backpack", 15)
	couriers = append(couriers, courier1)

	location2, _ := kernel.NewLocation(7, 2)
	courier2, _ := courier.NewCourier(kernel.NewUUID(), "Bob", 5, location2)
	courier2.AddStoragePlace("Large Box", 25)
	courier2.AddStoragePlace("Side Bag", 10)
	couriers = append(couriers, courier2)

	location3, _ := kernel.NewLocation(10, 10)
	courier3, _ := courier.NewCourier(kernel.NewUUID(), "Charlie", 2, location3)
	courier3.AddStoragePlace("Small Bag", 8)
	couriers = append(couriers, courier3)

	return couriers
}

func (suite *GetAllCouriersQueryHandlerTestSuite) saveCouriers(couriers []*courier.Courier) {
	repo := courierrepo.NewGormCourierRepository(suite.db, &mockAggregateTracker{})
	for _, c := range couriers {
		err := repo.Add(context.Background(), c)
		suite.Require().NoError(err)
	}
}

func (suite *GetAllCouriersQueryHandlerTestSuite) createAndSaveLargeCourierSet() {
	repo := courierrepo.NewGormCourierRepository(suite.db, &mockAggregateTracker{})
	for i := range 50 {
		location, _ := kernel.NewRandomLocation()
		courier, _ := courier.NewCourier(
			kernel.NewUUID(),
			"Courier",
			3,
			location,
		)
		err := repo.Add(context.Background(), courier)
		suite.Require().NoError(err)
		_ = i
	}
}

func TestGetAllCouriersQueryHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GetAllCouriersQueryHandlerTestSuite))
}

// mockAggregateTracker implements ports.AggregateTracker for test purposes.
// It's a no-op implementation since we don't need aggregate tracking in query tests.
type mockAggregateTracker struct{}

func (m *mockAggregateTracker) TrackAggregate(_ kernel.UUID, _ any) {
	// No-op for query tests
}

func (m *mockAggregateTracker) GetTrackedAggregates() []any {
	return []any{}
}
