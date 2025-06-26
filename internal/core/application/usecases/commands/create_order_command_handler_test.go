package commands_test

import (
	"context"
	"errors"
	"testing"

	"delivery/internal/core/application/usecases/commands"
	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/core/domain/model/order"
	"delivery/internal/core/ports"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockOrderRepository struct{ mock.Mock }

func (m *MockOrderRepository) Add(ctx context.Context, o *order.Order) error {
	args := m.Called(ctx, o)
	return args.Error(0)
}
func (m *MockOrderRepository) Update(_ context.Context, _ *order.Order) error { return nil }
func (m *MockOrderRepository) Get(_ context.Context, _ kernel.UUID) (*order.Order, error) {
	return nil, errors.New("not implemented in mock")
}
func (m *MockOrderRepository) GetFirstInCreatedStatus(_ context.Context) (*order.Order, error) {
	return nil, errors.New("not implemented in mock")
}
func (m *MockOrderRepository) GetAllInAssignedStatus(_ context.Context) ([]*order.Order, error) {
	return nil, errors.New("not implemented in mock")
}

type MockOrderUoW struct{ mock.Mock }

func (m *MockOrderUoW) Begin(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
func (m *MockOrderUoW) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
func (m *MockOrderUoW) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockOrderUoW) OrderRepository() ports.OrderRepository {
	args := m.Called()
	return args.Get(0).(ports.OrderRepository)
}

type MockOrderUoWFactory struct{ mock.Mock }

func (m *MockOrderUoWFactory) Create() commands.OrderUoW {
	args := m.Called()
	return args.Get(0).(commands.OrderUoW)
}

func TestCreateOrderCommandHandler_Handle_Success(t *testing.T) {
	ctx := t.Context()
	id := kernel.NewUUID()
	cmd, _ := commands.NewCreateOrderCommand(id, "Main St", 10)

	repo := new(MockOrderRepository)
	uow := new(MockOrderUoW)
	mock.InOrder(
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("OrderRepository").Return(repo).Once(),
		repo.On("Add", mock.Anything, mock.AnythingOfType("*order.Order")).Return(nil).Once(),
		uow.On("Commit", ctx).Return(nil).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	factory := new(MockOrderUoWFactory)
	factory.On("Create").Return(uow).Once()

	h := commands.NewCreateOrderCommandHandler(factory)
	err := h.Handle(ctx, cmd)
	require.NoError(t, err)
	repo.AssertExpectations(t)
	uow.AssertExpectations(t)
	factory.AssertExpectations(t)
}

func TestCreateOrderCommandHandler_Handle_ValidationError(t *testing.T) {
	ctx := t.Context()
	cmd := commands.CreateOrderCommand{} // not constructed properly
	factory := new(MockOrderUoWFactory)
	h := commands.NewCreateOrderCommandHandler(factory)
	err := h.Handle(ctx, cmd)
	require.Error(t, err)
}

func TestCreateOrderCommandHandler_Handle_BeginError(t *testing.T) {
	ctx := t.Context()
	id := kernel.NewUUID()
	cmd, _ := commands.NewCreateOrderCommand(id, "Main St", 10)

	uow := new(MockOrderUoW)
	factory := new(MockOrderUoWFactory)
	mock.InOrder(
		factory.On("Create").Return(uow).Once(),
		uow.On("Begin", ctx).Return(errors.New("begin error")).Once(),
	)

	h := commands.NewCreateOrderCommandHandler(factory)
	err := h.Handle(ctx, cmd)
	require.Error(t, err)
}

func TestCreateOrderCommandHandler_Handle_AddError(t *testing.T) {
	ctx := t.Context()
	id := kernel.NewUUID()
	cmd, _ := commands.NewCreateOrderCommand(id, "Main St", 10)

	repo := new(MockOrderRepository)
	uow := new(MockOrderUoW)
	mock.InOrder(
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("OrderRepository").Return(repo).Once(),
		repo.On("Add", mock.Anything, mock.AnythingOfType("*order.Order")).Return(errors.New("add error")).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	factory := new(MockOrderUoWFactory)
	factory.On("Create").Return(uow).Once()

	h := commands.NewCreateOrderCommandHandler(factory)
	err := h.Handle(ctx, cmd)
	require.Error(t, err)
	repo.AssertExpectations(t)
	uow.AssertExpectations(t)
	factory.AssertExpectations(t)
}

func TestCreateOrderCommandHandler_Handle_CommitError(t *testing.T) {
	ctx := t.Context()
	id := kernel.NewUUID()
	cmd, _ := commands.NewCreateOrderCommand(id, "Main St", 10)

	repo := new(MockOrderRepository)
	uow := new(MockOrderUoW)
	mock.InOrder(
		uow.On("Begin", ctx).Return(nil).Once(),
		uow.On("OrderRepository").Return(repo).Once(),
		repo.On("Add", mock.Anything, mock.AnythingOfType("*order.Order")).Return(nil).Once(),
		uow.On("Commit", ctx).Return(errors.New("commit error")).Once(),
		uow.On("Rollback", ctx).Return(nil).Once(),
	)

	factory := new(MockOrderUoWFactory)
	factory.On("Create").Return(uow).Once()

	h := commands.NewCreateOrderCommandHandler(factory)
	err := h.Handle(ctx, cmd)
	require.Error(t, err)
	repo.AssertExpectations(t)
	uow.AssertExpectations(t)
	factory.AssertExpectations(t)
}
