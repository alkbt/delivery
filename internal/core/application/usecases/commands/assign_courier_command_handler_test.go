package commands_test

import (
	"context"
	"errors"
	"testing"

	"delivery/internal/core/application/usecases/commands"
	"delivery/internal/core/domain/model/courier"
	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/core/domain/model/order"
	"delivery/internal/core/domain/services"
	"delivery/internal/core/ports"
	"delivery/internal/pkg/errs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockAssignCourierRepository struct{ mock.Mock }

func (m *MockAssignCourierRepository) Add(ctx context.Context, c *courier.Courier) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *MockAssignCourierRepository) Update(ctx context.Context, c *courier.Courier) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *MockAssignCourierRepository) Get(ctx context.Context, id kernel.UUID) (*courier.Courier, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*courier.Courier), args.Error(1)
}

func (m *MockAssignCourierRepository) GetAllFree(ctx context.Context) ([]*courier.Courier, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*courier.Courier), args.Error(1)
}

type MockAssignOrderRepository struct{ mock.Mock }

func (m *MockAssignOrderRepository) Add(ctx context.Context, o *order.Order) error {
	args := m.Called(ctx, o)
	return args.Error(0)
}

func (m *MockAssignOrderRepository) Update(ctx context.Context, o *order.Order) error {
	args := m.Called(ctx, o)
	return args.Error(0)
}

func (m *MockAssignOrderRepository) Get(ctx context.Context, id kernel.UUID) (*order.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*order.Order), args.Error(1)
}

func (m *MockAssignOrderRepository) GetFirstInCreatedStatus(ctx context.Context) (*order.Order, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*order.Order), args.Error(1)
}

func (m *MockAssignOrderRepository) GetAllInAssignedStatus(ctx context.Context) ([]*order.Order, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*order.Order), args.Error(1)
}

type MockAssignUoW struct{ mock.Mock }

func (m *MockAssignUoW) Begin(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockAssignUoW) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockAssignUoW) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockAssignUoW) OrderRepository() ports.OrderRepository {
	args := m.Called()
	return args.Get(0).(ports.OrderRepository)
}

func (m *MockAssignUoW) CourierRepository() ports.CourierRepository {
	args := m.Called()
	return args.Get(0).(ports.CourierRepository)
}

type MockAssignUoWFactory struct{ mock.Mock }

func (m *MockAssignUoWFactory) Create() commands.UoW {
	args := m.Called()
	return args.Get(0).(commands.UoW)
}

func TestAssignCourierCommandHandler_Handle_Success(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewAssignCourierCommand()

	// Create test data
	orderID := kernel.NewUUID()
	courierID := kernel.NewUUID()
	location, _ := kernel.NewLocation(5, 7)

	testOrder, _ := order.NewOrder(orderID, location, 10)
	testCourier, _ := courier.NewCourier(courierID, "John Doe", 3, location)
	testCouriers := []*courier.Courier{testCourier}

	// Setup mocks
	orderRepo := new(MockAssignOrderRepository)
	courierRepo := new(MockAssignCourierRepository)
	uow := new(MockAssignUoW)

	mock.InOrder(
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetFirstInCreatedStatus", ctx).Return(testOrder, nil).Once(),
		courierRepo.On("GetAllFree", ctx).Return(testCouriers, nil).Once(),
		orderRepo.On("Update", ctx, mock.AnythingOfType("*order.Order")).Return(nil).Once(),
		courierRepo.On("Update", ctx, mock.AnythingOfType("*courier.Courier")).Return(nil).Once(),
		uow.On("Commit", ctx).Return(nil).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	factory := new(MockAssignUoWFactory)
	factory.On("Create").Return(uow).Once()

	handler := commands.NewAssignCourierCommandHandler(factory)
	err := handler.Handle(ctx, cmd)

	require.NoError(t, err)
	orderRepo.AssertExpectations(t)
	courierRepo.AssertExpectations(t)
	uow.AssertExpectations(t)
	factory.AssertExpectations(t)
}

func TestAssignCourierCommandHandler_Handle_ValidationError(t *testing.T) {
	ctx := t.Context()
	cmd := commands.AssignCourierCommand{} // not constructed properly

	factory := new(MockAssignUoWFactory)
	handler := commands.NewAssignCourierCommandHandler(factory)
	err := handler.Handle(ctx, cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, commands.ErrAssignCourierCommandIsNotConstructed)
	factory.AssertNotCalled(t, "Create")
}

func TestAssignCourierCommandHandler_Handle_BeginError(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewAssignCourierCommand()

	uow := new(MockAssignUoW)
	factory := new(MockAssignUoWFactory)

	mock.InOrder(
		factory.On("Create").Return(uow).Once(),
		uow.On("Begin", ctx).Return(errors.New("begin error")).Once(),
	)

	handler := commands.NewAssignCourierCommandHandler(factory)
	err := handler.Handle(ctx, cmd)

	require.Error(t, err)
	require.EqualError(t, err, "begin error")
}

func TestAssignCourierCommandHandler_Handle_NoOrderFound(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewAssignCourierCommand()

	orderRepo := new(MockAssignOrderRepository)
	courierRepo := new(MockAssignCourierRepository)
	uow := new(MockAssignUoW)

	mock.InOrder(
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetFirstInCreatedStatus", ctx).Return(nil, errs.ErrObjectNotFound).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	factory := new(MockAssignUoWFactory)
	factory.On("Create").Return(uow).Once()

	handler := commands.NewAssignCourierCommandHandler(factory)
	err := handler.Handle(ctx, cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, commands.ErrNoOrderFound)
}

func TestAssignCourierCommandHandler_Handle_GetOrderError(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewAssignCourierCommand()

	orderRepo := new(MockAssignOrderRepository)
	courierRepo := new(MockAssignCourierRepository)
	uow := new(MockAssignUoW)

	mock.InOrder(
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetFirstInCreatedStatus", ctx).Return(nil, errors.New("database error")).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	factory := new(MockAssignUoWFactory)
	factory.On("Create").Return(uow).Once()

	handler := commands.NewAssignCourierCommandHandler(factory)
	err := handler.Handle(ctx, cmd)

	require.Error(t, err)
	require.EqualError(t, err, "database error")
}

func TestAssignCourierCommandHandler_Handle_NoFreeCouriers(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewAssignCourierCommand()

	orderID := kernel.NewUUID()
	location, _ := kernel.NewLocation(5, 7)
	testOrder, _ := order.NewOrder(orderID, location, 10)

	orderRepo := new(MockAssignOrderRepository)
	courierRepo := new(MockAssignCourierRepository)
	uow := new(MockAssignUoW)

	mock.InOrder(
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetFirstInCreatedStatus", ctx).Return(testOrder, nil).Once(),
		courierRepo.On("GetAllFree", ctx).Return([]*courier.Courier{}, nil).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	factory := new(MockAssignUoWFactory)
	factory.On("Create").Return(uow).Once()

	handler := commands.NewAssignCourierCommandHandler(factory)
	err := handler.Handle(ctx, cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, commands.ErrNoFreeCouriersFound)
}

func TestAssignCourierCommandHandler_Handle_GetCouriersError(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewAssignCourierCommand()

	orderID := kernel.NewUUID()
	location, _ := kernel.NewLocation(5, 7)
	testOrder, _ := order.NewOrder(orderID, location, 10)

	orderRepo := new(MockAssignOrderRepository)
	courierRepo := new(MockAssignCourierRepository)
	uow := new(MockAssignUoW)

	mock.InOrder(
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetFirstInCreatedStatus", ctx).Return(testOrder, nil).Once(),
		courierRepo.On("GetAllFree", ctx).Return(nil, errors.New("database error")).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	factory := new(MockAssignUoWFactory)
	factory.On("Create").Return(uow).Once()

	handler := commands.NewAssignCourierCommandHandler(factory)
	err := handler.Handle(ctx, cmd)

	require.Error(t, err)
	require.EqualError(t, err, "database error")
}

func TestAssignCourierCommandHandler_Handle_DispatchError(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewAssignCourierCommand()

	orderID := kernel.NewUUID()
	location, _ := kernel.NewLocation(5, 7)
	testOrder, _ := order.NewOrder(orderID, location, 1000) // Too heavy

	courierID := kernel.NewUUID()
	testCourier, _ := courier.NewCourier(courierID, "John Doe", 3, location)
	testCouriers := []*courier.Courier{testCourier}

	orderRepo := new(MockAssignOrderRepository)
	courierRepo := new(MockAssignCourierRepository)
	uow := new(MockAssignUoW)

	mock.InOrder(
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetFirstInCreatedStatus", ctx).Return(testOrder, nil).Once(),
		courierRepo.On("GetAllFree", ctx).Return(testCouriers, nil).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	factory := new(MockAssignUoWFactory)
	factory.On("Create").Return(uow).Once()

	handler := commands.NewAssignCourierCommandHandler(factory)
	err := handler.Handle(ctx, cmd)

	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrCourierNotFound)
}

func TestAssignCourierCommandHandler_Handle_UpdateOrderError(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewAssignCourierCommand()

	orderID := kernel.NewUUID()
	courierID := kernel.NewUUID()
	location, _ := kernel.NewLocation(5, 7)

	testOrder, _ := order.NewOrder(orderID, location, 10)
	testCourier, _ := courier.NewCourier(courierID, "John Doe", 3, location)
	testCouriers := []*courier.Courier{testCourier}

	orderRepo := new(MockAssignOrderRepository)
	courierRepo := new(MockAssignCourierRepository)
	uow := new(MockAssignUoW)

	mock.InOrder(
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetFirstInCreatedStatus", ctx).Return(testOrder, nil).Once(),
		courierRepo.On("GetAllFree", ctx).Return(testCouriers, nil).Once(),
		orderRepo.On("Update", ctx, mock.AnythingOfType("*order.Order")).Return(errors.New("update error")).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	factory := new(MockAssignUoWFactory)
	factory.On("Create").Return(uow).Once()

	handler := commands.NewAssignCourierCommandHandler(factory)
	err := handler.Handle(ctx, cmd)

	require.Error(t, err)
	require.EqualError(t, err, "update error")
}

func TestAssignCourierCommandHandler_Handle_UpdateCourierError(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewAssignCourierCommand()

	orderID := kernel.NewUUID()
	courierID := kernel.NewUUID()
	location, _ := kernel.NewLocation(5, 7)

	testOrder, _ := order.NewOrder(orderID, location, 10)
	testCourier, _ := courier.NewCourier(courierID, "John Doe", 3, location)
	testCouriers := []*courier.Courier{testCourier}

	orderRepo := new(MockAssignOrderRepository)
	courierRepo := new(MockAssignCourierRepository)
	uow := new(MockAssignUoW)

	mock.InOrder(
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetFirstInCreatedStatus", ctx).Return(testOrder, nil).Once(),
		courierRepo.On("GetAllFree", ctx).Return(testCouriers, nil).Once(),
		orderRepo.On("Update", ctx, mock.AnythingOfType("*order.Order")).Return(nil).Once(),
		courierRepo.On("Update", ctx, mock.AnythingOfType("*courier.Courier")).
			Return(errors.New("update error")).
			Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	factory := new(MockAssignUoWFactory)
	factory.On("Create").Return(uow).Once()

	handler := commands.NewAssignCourierCommandHandler(factory)
	err := handler.Handle(ctx, cmd)

	require.Error(t, err)
	require.EqualError(t, err, "update error")
}

func TestAssignCourierCommandHandler_Handle_CommitError(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewAssignCourierCommand()

	orderID := kernel.NewUUID()
	courierID := kernel.NewUUID()
	location, _ := kernel.NewLocation(5, 7)

	testOrder, _ := order.NewOrder(orderID, location, 10)
	testCourier, _ := courier.NewCourier(courierID, "John Doe", 3, location)
	testCouriers := []*courier.Courier{testCourier}

	orderRepo := new(MockAssignOrderRepository)
	courierRepo := new(MockAssignCourierRepository)
	uow := new(MockAssignUoW)

	mock.InOrder(
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetFirstInCreatedStatus", ctx).Return(testOrder, nil).Once(),
		courierRepo.On("GetAllFree", ctx).Return(testCouriers, nil).Once(),
		orderRepo.On("Update", ctx, mock.AnythingOfType("*order.Order")).Return(nil).Once(),
		courierRepo.On("Update", ctx, mock.AnythingOfType("*courier.Courier")).Return(nil).Once(),
		uow.On("Commit", ctx).Return(errors.New("commit error")).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	factory := new(MockAssignUoWFactory)
	factory.On("Create").Return(uow).Once()

	handler := commands.NewAssignCourierCommandHandler(factory)
	err := handler.Handle(ctx, cmd)

	require.Error(t, err)
	require.EqualError(t, err, "commit error")
}

func TestAssignCourierCommandHandler_Handle_MultipleCouriers(t *testing.T) {
	ctx := t.Context()
	cmd := commands.NewAssignCourierCommand()

	orderID := kernel.NewUUID()
	location, _ := kernel.NewLocation(5, 7)
	testOrder, _ := order.NewOrder(orderID, location, 10)

	// Create multiple couriers at different locations
	courier1ID := kernel.NewUUID()
	courier2ID := kernel.NewUUID()
	courier3ID := kernel.NewUUID()

	// Create couriers at different locations for testing distance calculation
	// Courier 2 will be closest to the order at (5,7)
	loc1, _ := kernel.NewLocation(1, 1)   // Far away (corner)
	loc2, _ := kernel.NewLocation(6, 7)   // Very close to order
	loc3, _ := kernel.NewLocation(10, 10) // Medium distance (opposite corner)

	testCourier1, err := courier.NewCourier(courier1ID, "John Doe", 3, loc1)
	require.NoError(t, err)
	testCourier2, err := courier.NewCourier(courier2ID, "Jane Smith", 3, loc2)
	require.NoError(t, err)
	testCourier3, err := courier.NewCourier(courier3ID, "Bob Wilson", 3, loc3)
	require.NoError(t, err)

	testCouriers := []*courier.Courier{testCourier1, testCourier2, testCourier3}

	orderRepo := new(MockAssignOrderRepository)
	courierRepo := new(MockAssignCourierRepository)
	uow := new(MockAssignUoW)

	mock.InOrder(
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("CourierRepository").Return(courierRepo).Once(),
		uow.On("OrderRepository").Return(orderRepo).Once(),
		orderRepo.On("GetFirstInCreatedStatus", ctx).Return(testOrder, nil).Once(),
		courierRepo.On("GetAllFree", ctx).Return(testCouriers, nil).Once(),
		orderRepo.On("Update", ctx, mock.AnythingOfType("*order.Order")).Return(nil).Once(),
		courierRepo.On("Update", ctx, mock.AnythingOfType("*courier.Courier")).Return(nil).Once(),
		uow.On("Commit", ctx).Return(nil).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	factory := new(MockAssignUoWFactory)
	factory.On("Create").Return(uow).Once()

	handler := commands.NewAssignCourierCommandHandler(factory)
	err = handler.Handle(ctx, cmd)

	require.NoError(t, err)

	// Verify that the courier with the nearest location was selected
	updateCall := courierRepo.Calls[1]
	updatedCourier := updateCall.Arguments[1].(*courier.Courier)
	assert.Equal(t, courier2ID, updatedCourier.ID())
}
