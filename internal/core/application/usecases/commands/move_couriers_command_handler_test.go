package commands_test

import (
	"context"
	"errors"
	"testing"

	"delivery/internal/core/application/usecases/commands"
	"delivery/internal/core/domain/model/courier"
	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/core/domain/model/order"
	"delivery/internal/core/ports"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MoveCourierRepo struct{ mock.Mock }

func (m *MoveCourierRepo) Add(ctx context.Context, c *courier.Courier) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *MoveCourierRepo) Update(ctx context.Context, c *courier.Courier) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *MoveCourierRepo) Get(ctx context.Context, id kernel.UUID) (*courier.Courier, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*courier.Courier), args.Error(1)
}

func (m *MoveCourierRepo) GetAllFree(ctx context.Context) ([]*courier.Courier, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*courier.Courier), args.Error(1)
}

type MoveOrderRepo struct{ mock.Mock }

func (m *MoveOrderRepo) Add(ctx context.Context, o *order.Order) error {
	args := m.Called(ctx, o)
	return args.Error(0)
}

func (m *MoveOrderRepo) Update(ctx context.Context, o *order.Order) error {
	args := m.Called(ctx, o)
	return args.Error(0)
}

func (m *MoveOrderRepo) Get(ctx context.Context, id kernel.UUID) (*order.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*order.Order), args.Error(1)
}

func (m *MoveOrderRepo) GetFirstInCreatedStatus(ctx context.Context) (*order.Order, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*order.Order), args.Error(1)
}

func (m *MoveOrderRepo) GetAllInAssignedStatus(ctx context.Context) ([]*order.Order, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*order.Order), args.Error(1)
}

type MoveUnitOfWork struct{ mock.Mock }

func (m *MoveUnitOfWork) Begin(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MoveUnitOfWork) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MoveUnitOfWork) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MoveUnitOfWork) CourierRepository() ports.CourierRepository {
	args := m.Called()
	return args.Get(0).(ports.CourierRepository)
}

func (m *MoveUnitOfWork) OrderRepository() ports.OrderRepository {
	args := m.Called()
	return args.Get(0).(ports.OrderRepository)
}

type MoveUoWFactory struct{ mock.Mock }

func (m *MoveUoWFactory) Create() commands.UoW {
	args := m.Called()
	return args.Get(0).(commands.UoW)
}

func createTestOrderWithCourier(
	courierID kernel.UUID,
	orderLocation kernel.Location,
	courierLocation kernel.Location,
) (*order.Order, *courier.Courier, error) {
	// Create order
	orderID := kernel.NewUUID()
	testOrder, err := order.NewOrder(orderID, orderLocation, 5)
	if err != nil {
		return nil, nil, err
	}

	// Assign courier to order
	err = testOrder.Assign(courierID)
	if err != nil {
		return nil, nil, err
	}

	// Create courier
	testCourier, err := courier.NewCourier(courierID, "Test Courier", 2, courierLocation)
	if err != nil {
		return nil, nil, err
	}

	return testOrder, testCourier, nil
}

func TestMoveCouriersCommandHandler_Handle_Success(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewMoveCouriersCommand()

	// Create test data
	courierID := kernel.NewUUID()
	orderLocation, _ := kernel.NewLocation(5, 5)
	courierLocation, _ := kernel.NewLocation(3, 3)
	testOrder, testCourier, err := createTestOrderWithCourier(courierID, orderLocation, courierLocation)
	require.NoError(t, err)

	// Ensure courier has the order in storage for successful completion
	err = testCourier.TakeOrder(testOrder)
	require.NoError(t, err)

	// Set up mocks
	courierRepo := new(MoveCourierRepo)
	orderRepo := new(MoveOrderRepo)
	uow := new(MoveUnitOfWork)
	factory := new(MoveUoWFactory)

	mock.InOrder(
		factory.On("Create").Return(uow).Once(),
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetAllInAssignedStatus", ctx).Return([]*order.Order{testOrder}, nil).Once(),
		courierRepo.On("Get", ctx, courierID).Return(testCourier, nil).Once(),
		orderRepo.On("Update", ctx, testOrder).Return(nil).Once(),
		courierRepo.On("Update", ctx, testCourier).Return(nil).Once(),
		uow.On("Commit", ctx).Return(nil).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	// Act
	handler := commands.NewMoveCouriersCommandHandler(factory)
	err = handler.Handle(ctx, cmd)

	// Assert
	require.NoError(t, err)
	factory.AssertExpectations(t)
	uow.AssertExpectations(t)
	orderRepo.AssertExpectations(t)
	courierRepo.AssertExpectations(t)
}

func TestMoveCouriersCommandHandler_Handle_ValidationError(t *testing.T) {
	ctx := t.Context()
	cmd := commands.MoveCouriersCommand{} // not constructed properly
	factory := new(MoveUoWFactory)

	handler := commands.NewMoveCouriersCommandHandler(factory)
	err := handler.Handle(ctx, cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be created via NewMoveCouriersCommand constructor")
}

func TestMoveCouriersCommandHandler_Handle_BeginError(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewMoveCouriersCommand()

	uow := new(MoveUnitOfWork)
	factory := new(MoveUoWFactory)

	mock.InOrder(
		factory.On("Create").Return(uow).Once(),
		uow.On("Begin", ctx).Return(errors.New("begin error")).Once(),
	)

	handler := commands.NewMoveCouriersCommandHandler(factory)
	err := handler.Handle(ctx, cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "begin error")
	factory.AssertExpectations(t)
	uow.AssertExpectations(t)
}

func TestMoveCouriersCommandHandler_Handle_GetAllInAssignedStatusError(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewMoveCouriersCommand()

	courierRepo := new(MoveCourierRepo)
	orderRepo := new(MoveOrderRepo)
	uow := new(MoveUnitOfWork)
	factory := new(MoveUoWFactory)

	mock.InOrder(
		factory.On("Create").Return(uow).Once(),
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetAllInAssignedStatus", ctx).Return(nil, errors.New("repository error")).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	handler := commands.NewMoveCouriersCommandHandler(factory)
	err := handler.Handle(ctx, cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "repository error")
	factory.AssertExpectations(t)
	uow.AssertExpectations(t)
	orderRepo.AssertExpectations(t)
}

func TestMoveCouriersCommandHandler_Handle_GetCourierError(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewMoveCouriersCommand()

	// Create test data
	courierID := kernel.NewUUID()
	orderLocation, _ := kernel.NewLocation(5, 5)
	courierLocation, _ := kernel.NewLocation(3, 3)
	testOrder, _, err := createTestOrderWithCourier(courierID, orderLocation, courierLocation)
	require.NoError(t, err)

	courierRepo := new(MoveCourierRepo)
	orderRepo := new(MoveOrderRepo)
	uow := new(MoveUnitOfWork)
	factory := new(MoveUoWFactory)

	mock.InOrder(
		factory.On("Create").Return(uow).Once(),
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetAllInAssignedStatus", ctx).Return([]*order.Order{testOrder}, nil).Once(),
		courierRepo.On("Get", ctx, courierID).Return(nil, errors.New("courier not found")).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	handler := commands.NewMoveCouriersCommandHandler(factory)
	err = handler.Handle(ctx, cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "courier not found")
	factory.AssertExpectations(t)
	uow.AssertExpectations(t)
	orderRepo.AssertExpectations(t)
	courierRepo.AssertExpectations(t)
}

func TestMoveCouriersCommandHandler_Handle_CourierCompleteOrderError(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewMoveCouriersCommand()

	// Create test data - courier reaches order location but doesn't have the order in storage
	courierID := kernel.NewUUID()
	orderLocation, _ := kernel.NewLocation(2, 2)
	courierLocation, _ := kernel.NewLocation(2, 2) // Same location
	testOrder, testCourier, err := createTestOrderWithCourier(courierID, orderLocation, courierLocation)
	require.NoError(t, err)

	courierRepo := new(MoveCourierRepo)
	orderRepo := new(MoveOrderRepo)
	uow := new(MoveUnitOfWork)
	factory := new(MoveUoWFactory)

	mock.InOrder(
		factory.On("Create").Return(uow).Once(),
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetAllInAssignedStatus", ctx).Return([]*order.Order{testOrder}, nil).Once(),
		courierRepo.On("Get", ctx, courierID).Return(testCourier, nil).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	handler := commands.NewMoveCouriersCommandHandler(factory)
	err = handler.Handle(ctx, cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "storage place not found")
	factory.AssertExpectations(t)
	uow.AssertExpectations(t)
	orderRepo.AssertExpectations(t)
	courierRepo.AssertExpectations(t)
}

func TestMoveCouriersCommandHandler_Handle_OrderUpdateError(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewMoveCouriersCommand()

	// Create test data
	courierID := kernel.NewUUID()
	orderLocation, _ := kernel.NewLocation(2, 2)
	courierLocation, _ := kernel.NewLocation(2, 2) // Same location
	testOrder, testCourier, err := createTestOrderWithCourier(courierID, orderLocation, courierLocation)
	require.NoError(t, err)

	// We need to ensure the courier has the order in storage for the test to work properly
	err = testCourier.TakeOrder(testOrder)
	require.NoError(t, err)

	courierRepo := new(MoveCourierRepo)
	orderRepo := new(MoveOrderRepo)
	uow := new(MoveUnitOfWork)
	factory := new(MoveUoWFactory)

	mock.InOrder(
		factory.On("Create").Return(uow).Once(),
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetAllInAssignedStatus", ctx).Return([]*order.Order{testOrder}, nil).Once(),
		courierRepo.On("Get", ctx, courierID).Return(testCourier, nil).Once(),
		orderRepo.On("Update", ctx, testOrder).Return(errors.New("order update error")).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	handler := commands.NewMoveCouriersCommandHandler(factory)
	err = handler.Handle(ctx, cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "order update error")
	factory.AssertExpectations(t)
	uow.AssertExpectations(t)
	orderRepo.AssertExpectations(t)
	courierRepo.AssertExpectations(t)
}

func TestMoveCouriersCommandHandler_Handle_CourierUpdateError(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewMoveCouriersCommand()

	// Create test data
	courierID := kernel.NewUUID()
	orderLocation, _ := kernel.NewLocation(2, 2)
	courierLocation, _ := kernel.NewLocation(2, 2) // Same location
	testOrder, testCourier, err := createTestOrderWithCourier(courierID, orderLocation, courierLocation)
	require.NoError(t, err)

	// We need to ensure the courier has the order in storage for the test to work properly
	err = testCourier.TakeOrder(testOrder)
	require.NoError(t, err)

	courierRepo := new(MoveCourierRepo)
	orderRepo := new(MoveOrderRepo)
	uow := new(MoveUnitOfWork)
	factory := new(MoveUoWFactory)

	mock.InOrder(
		factory.On("Create").Return(uow).Once(),
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetAllInAssignedStatus", ctx).Return([]*order.Order{testOrder}, nil).Once(),
		courierRepo.On("Get", ctx, courierID).Return(testCourier, nil).Once(),
		orderRepo.On("Update", ctx, testOrder).Return(nil).Once(),
		courierRepo.On("Update", ctx, testCourier).Return(errors.New("courier update error")).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	handler := commands.NewMoveCouriersCommandHandler(factory)
	err = handler.Handle(ctx, cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "courier update error")
	factory.AssertExpectations(t)
	uow.AssertExpectations(t)
	orderRepo.AssertExpectations(t)
	courierRepo.AssertExpectations(t)
}

func TestMoveCouriersCommandHandler_Handle_CommitError(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewMoveCouriersCommand()

	// Create test data
	courierID := kernel.NewUUID()
	orderLocation, _ := kernel.NewLocation(2, 2)
	courierLocation, _ := kernel.NewLocation(2, 2)
	testOrder, testCourier, err := createTestOrderWithCourier(courierID, orderLocation, courierLocation)
	require.NoError(t, err)

	// We need to ensure the courier has the order in storage for the test to work properly
	err = testCourier.TakeOrder(testOrder)
	require.NoError(t, err)

	courierRepo := new(MoveCourierRepo)
	orderRepo := new(MoveOrderRepo)
	uow := new(MoveUnitOfWork)
	factory := new(MoveUoWFactory)

	mock.InOrder(
		factory.On("Create").Return(uow).Once(),
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetAllInAssignedStatus", ctx).Return([]*order.Order{testOrder}, nil).Once(),
		courierRepo.On("Get", ctx, courierID).Return(testCourier, nil).Once(),
		orderRepo.On("Update", ctx, testOrder).Return(nil).Once(),
		courierRepo.On("Update", ctx, testCourier).Return(nil).Once(),
		uow.On("Commit", ctx).Return(errors.New("commit error")).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	handler := commands.NewMoveCouriersCommandHandler(factory)
	err = handler.Handle(ctx, cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "commit error")
	factory.AssertExpectations(t)
	uow.AssertExpectations(t)
	orderRepo.AssertExpectations(t)
	courierRepo.AssertExpectations(t)
}

func TestMoveCouriersCommandHandler_Handle_NoOrdersAssigned(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewMoveCouriersCommand()

	courierRepo := new(MoveCourierRepo)
	orderRepo := new(MoveOrderRepo)
	uow := new(MoveUnitOfWork)
	factory := new(MoveUoWFactory)

	mock.InOrder(
		factory.On("Create").Return(uow).Once(),
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetAllInAssignedStatus", ctx).Return([]*order.Order{}, nil).Once(),
		uow.On("Commit", ctx).Return(nil).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	handler := commands.NewMoveCouriersCommandHandler(factory)
	err := handler.Handle(ctx, cmd)

	require.NoError(t, err)
	factory.AssertExpectations(t)
	uow.AssertExpectations(t)
	orderRepo.AssertExpectations(t)
}

func TestMoveCouriersCommandHandler_Handle_MultipleOrders(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewMoveCouriersCommand()

	// Create test data for multiple orders
	courierID1 := kernel.NewUUID()
	courierID2 := kernel.NewUUID()
	orderLocation1, _ := kernel.NewLocation(1, 1)
	orderLocation2, _ := kernel.NewLocation(2, 2)
	courierLocation1, _ := kernel.NewLocation(1, 1)
	courierLocation2, _ := kernel.NewLocation(2, 2)

	testOrder1, testCourier1, err := createTestOrderWithCourier(courierID1, orderLocation1, courierLocation1)
	require.NoError(t, err)
	testOrder2, testCourier2, err := createTestOrderWithCourier(courierID2, orderLocation2, courierLocation2)
	require.NoError(t, err)

	// Ensure couriers have the orders in their storage for successful completion
	err = testCourier1.TakeOrder(testOrder1)
	require.NoError(t, err)
	err = testCourier2.TakeOrder(testOrder2)
	require.NoError(t, err)

	courierRepo := new(MoveCourierRepo)
	orderRepo := new(MoveOrderRepo)
	uow := new(MoveUnitOfWork)
	factory := new(MoveUoWFactory)

	mock.InOrder(
		factory.On("Create").Return(uow).Once(),
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetAllInAssignedStatus", ctx).Return([]*order.Order{testOrder1, testOrder2}, nil).Once(),
		// First order processing
		courierRepo.On("Get", ctx, courierID1).Return(testCourier1, nil).Once(),
		orderRepo.On("Update", ctx, testOrder1).Return(nil).Once(),
		courierRepo.On("Update", ctx, testCourier1).Return(nil).Once(),
		// Second order processing
		courierRepo.On("Get", ctx, courierID2).Return(testCourier2, nil).Once(),
		orderRepo.On("Update", ctx, testOrder2).Return(nil).Once(),
		courierRepo.On("Update", ctx, testCourier2).Return(nil).Once(),
		uow.On("Commit", ctx).Return(nil).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	handler := commands.NewMoveCouriersCommandHandler(factory)
	err = handler.Handle(ctx, cmd)

	require.NoError(t, err)
	factory.AssertExpectations(t)
	uow.AssertExpectations(t)
	orderRepo.AssertExpectations(t)
	courierRepo.AssertExpectations(t)
}

func TestMoveCouriersCommandHandler_Handle_CourierMovesOneStepTowardDestination(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewMoveCouriersCommand()

	// Create test data where courier needs more than one step to reach destination
	courierID := kernel.NewUUID()
	orderLocation, _ := kernel.NewLocation(10, 10) // Order at (10, 10)
	courierLocation, _ := kernel.NewLocation(5, 5) // Courier starts at (5, 5)
	testOrder, testCourier, err := createTestOrderWithCourier(courierID, orderLocation, courierLocation)
	require.NoError(t, err)

	// Courier speed is 2 (from createTestOrderWithCourier), so it can only move 2 steps
	// Distance from (5,5) to (10,10) is 10 steps, so courier won't reach destination

	courierRepo := new(MoveCourierRepo)
	orderRepo := new(MoveOrderRepo)
	uow := new(MoveUnitOfWork)
	factory := new(MoveUoWFactory)

	mock.InOrder(
		factory.On("Create").Return(uow).Once(),
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetAllInAssignedStatus", ctx).Return([]*order.Order{testOrder}, nil).Once(),
		courierRepo.On("Get", ctx, courierID).Return(testCourier, nil).Once(),
		// Courier moves but doesn't reach destination - this is allowed and successful
		orderRepo.On("Update", ctx, testOrder).Return(nil).Once(),
		courierRepo.On("Update", ctx, testCourier).Return(nil).Once(),
		uow.On("Commit", ctx).Return(nil).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	handler := commands.NewMoveCouriersCommandHandler(factory)
	err = handler.Handle(ctx, cmd)

	// Should succeed - courier moved but didn't reach destination (partial movement is allowed)
	require.NoError(t, err)
	factory.AssertExpectations(t)
	uow.AssertExpectations(t)
	orderRepo.AssertExpectations(t)
	courierRepo.AssertExpectations(t)

	// Verify courier moved in the right direction but didn't reach destination
	expectedLocation, _ := kernel.NewLocation(7, 5) // Moved 2 steps horizontally (X-axis priority)
	actualLocation := testCourier.Location()
	isEqual, _ := actualLocation.IsEqual(expectedLocation)
	assert.True(t, isEqual, "Courier should have moved 2 steps toward destination")
}

func TestMoveCouriersCommandHandler_Handle_CourierMovesPartiallyMultipleSteps(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewMoveCouriersCommand()

	// Test case where courier moves partially in both X and Y directions
	courierID := kernel.NewUUID()
	orderLocation, _ := kernel.NewLocation(8, 7)   // Order at (8, 7)
	courierLocation, _ := kernel.NewLocation(5, 5) // Courier starts at (5, 5)
	testOrder, testCourier, err := createTestOrderWithCourier(courierID, orderLocation, courierLocation)
	require.NoError(t, err)

	// Distance is 3 (X) + 2 (Y) = 5 steps total
	// Courier speed is 2, so it will move 2 steps toward destination (prioritizing X-axis)

	courierRepo := new(MoveCourierRepo)
	orderRepo := new(MoveOrderRepo)
	uow := new(MoveUnitOfWork)
	factory := new(MoveUoWFactory)

	mock.InOrder(
		factory.On("Create").Return(uow).Once(),
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetAllInAssignedStatus", ctx).Return([]*order.Order{testOrder}, nil).Once(),
		courierRepo.On("Get", ctx, courierID).Return(testCourier, nil).Once(),
		// Courier moves but doesn't reach destination - this is allowed and successful
		orderRepo.On("Update", ctx, testOrder).Return(nil).Once(),
		courierRepo.On("Update", ctx, testCourier).Return(nil).Once(),
		uow.On("Commit", ctx).Return(nil).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	handler := commands.NewMoveCouriersCommandHandler(factory)
	err = handler.Handle(ctx, cmd)

	require.NoError(t, err) // Should succeed - courier moved but didn't reach destination
	factory.AssertExpectations(t)
	uow.AssertExpectations(t)
	orderRepo.AssertExpectations(t)
	courierRepo.AssertExpectations(t)

	// Verify courier moved 2 steps horizontally (X-axis priority)
	expectedLocation, _ := kernel.NewLocation(7, 5) // Moved 2 steps in X direction
	actualLocation := testCourier.Location()
	isEqual, _ := actualLocation.IsEqual(expectedLocation)
	assert.True(t, isEqual, "Courier should have moved 2 steps horizontally (X-axis priority)")
}

func TestMoveCouriersCommandHandler_Handle_CourierMovesExactlyToDestinationButHasNoOrder(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewMoveCouriersCommand()

	// Test case where courier reaches destination in one step but doesn't have the order in storage
	courierID := kernel.NewUUID()
	orderLocation, _ := kernel.NewLocation(7, 5)   // Order at (7, 5)
	courierLocation, _ := kernel.NewLocation(5, 5) // Courier starts at (5, 5)
	testOrder, testCourier, err := createTestOrderWithCourier(courierID, orderLocation, courierLocation)
	require.NoError(t, err)

	// Distance is 2 steps, courier speed is 2, so courier will reach destination
	// But courier doesn't have the order in storage, so completion should fail

	courierRepo := new(MoveCourierRepo)
	orderRepo := new(MoveOrderRepo)
	uow := new(MoveUnitOfWork)
	factory := new(MoveUoWFactory)

	mock.InOrder(
		factory.On("Create").Return(uow).Once(),
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetAllInAssignedStatus", ctx).Return([]*order.Order{testOrder}, nil).Once(),
		courierRepo.On("Get", ctx, courierID).Return(testCourier, nil).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	handler := commands.NewMoveCouriersCommandHandler(factory)
	err = handler.Handle(ctx, cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "storage place not found")
	factory.AssertExpectations(t)
	uow.AssertExpectations(t)
	orderRepo.AssertExpectations(t)
	courierRepo.AssertExpectations(t)

	// Verify courier reached the destination
	actualLocation := testCourier.Location()
	isEqual, _ := actualLocation.IsEqual(orderLocation)
	assert.True(t, isEqual, "Courier should have reached the order location")
}

func TestMoveCouriersCommandHandler_Handle_MultipleOrdersPartialMovement(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewMoveCouriersCommand()

	// Test case with multiple orders where couriers make partial movement
	courierID1 := kernel.NewUUID()
	courierID2 := kernel.NewUUID()
	orderLocation1, _ := kernel.NewLocation(10, 10) // Far destination
	orderLocation2, _ := kernel.NewLocation(3, 3)   // Close destination
	courierLocation1, _ := kernel.NewLocation(5, 5) // Far from destination
	courierLocation2, _ := kernel.NewLocation(2, 2) // Close to destination

	testOrder1, testCourier1, err := createTestOrderWithCourier(courierID1, orderLocation1, courierLocation1)
	require.NoError(t, err)
	testOrder2, testCourier2, err := createTestOrderWithCourier(courierID2, orderLocation2, courierLocation2)
	require.NoError(t, err)

	// Courier 2 has the order in storage and will complete it
	err = testCourier2.TakeOrder(testOrder2)
	require.NoError(t, err)

	courierRepo := new(MoveCourierRepo)
	orderRepo := new(MoveOrderRepo)
	uow := new(MoveUnitOfWork)
	factory := new(MoveUoWFactory)

	mock.InOrder(
		factory.On("Create").Return(uow).Once(),
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetAllInAssignedStatus", ctx).Return([]*order.Order{testOrder1, testOrder2}, nil).Once(),
		// First courier - moves but doesn't reach destination (partial movement is allowed)
		courierRepo.On("Get", ctx, courierID1).Return(testCourier1, nil).Once(),
		orderRepo.On("Update", ctx, testOrder1).Return(nil).Once(),
		courierRepo.On("Update", ctx, testCourier1).Return(nil).Once(),
		// Second courier - reaches destination and completes order
		courierRepo.On("Get", ctx, courierID2).Return(testCourier2, nil).Once(),
		orderRepo.On("Update", ctx, testOrder2).Return(nil).Once(),
		courierRepo.On("Update", ctx, testCourier2).Return(nil).Once(),
		uow.On("Commit", ctx).Return(nil).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	handler := commands.NewMoveCouriersCommandHandler(factory)
	err = handler.Handle(ctx, cmd)

	require.NoError(t, err) // Should succeed - both couriers processed successfully
	factory.AssertExpectations(t)
	uow.AssertExpectations(t)
	orderRepo.AssertExpectations(t)
	courierRepo.AssertExpectations(t)
}
