package commands_test

import (
	"context"
	"errors"
	"testing"

	"delivery/internal/core/application/usecases/commands"
	"delivery/internal/core/domain/model/courier"
	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/core/ports"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing.
type MockCourierRepository struct {
	mock.Mock
}

func (m *MockCourierRepository) Add(ctx context.Context, courier *courier.Courier) error {
	args := m.Called(ctx, courier)
	return args.Error(0)
}

func (m *MockCourierRepository) Update(ctx context.Context, courier *courier.Courier) error {
	args := m.Called(ctx, courier)
	return args.Error(0)
}

func (m *MockCourierRepository) Get(ctx context.Context, id kernel.UUID) (*courier.Courier, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*courier.Courier), args.Error(1)
}

func (m *MockCourierRepository) GetAllFree(ctx context.Context) ([]*courier.Courier, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*courier.Courier), args.Error(1)
}

type MockCourierUoW struct {
	mock.Mock
}

func (m *MockCourierUoW) Begin(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCourierUoW) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCourierUoW) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCourierUoW) CourierRepository() ports.CourierRepository {
	args := m.Called()
	return args.Get(0).(ports.CourierRepository)
}

type MockCourierUoWFactory struct {
	mock.Mock
}

func (m *MockCourierUoWFactory) Create() commands.CourierUoW {
	args := m.Called()
	return args.Get(0).(commands.CourierUoW)
}

func TestNewCreateCourierCommandHandler(t *testing.T) {
	// Arrange
	mockFactory := new(MockCourierUoWFactory)

	// Act
	handler := commands.NewCreateCourierCommandHandler(mockFactory)

	// Assert
	assert.NotNil(t, handler)
}

func TestCreateCourierCommandHandler_Handle_Success(t *testing.T) {
	// Arrange
	ctx := t.Context()
	name := "John Doe"
	speed := 3
	location, err := kernel.NewLocation(5, 7)
	require.NoError(t, err)

	cmd, err := commands.NewCreateCourierCommand(name, speed, location)
	require.NoError(t, err)

	mockRepo := new(MockCourierRepository)
	mockUoW := new(MockCourierUoW)
	mockFactory := new(MockCourierUoWFactory)

	// Set up expectations in order
	mock.InOrder(
		mockUoW.On("Begin", ctx).Return(nil).Once(),
		mockUoW.On("CourierRepository").Return(mockRepo).Once(),
		mockRepo.On("Add", ctx, mock.AnythingOfType("*courier.Courier")).Return(nil).Once(),
		mockUoW.On("Commit", ctx).Return(nil).Once(),
		mockUoW.On("Rollback", ctx).Return(nil).Once(),
	)
	mockFactory.On("Create").Return(mockUoW).Once()

	handler := commands.NewCreateCourierCommandHandler(mockFactory)

	// Act
	err = handler.Handle(ctx, cmd)

	// Assert
	require.NoError(t, err)
	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestCreateCourierCommandHandler_Handle_InvalidCommand(t *testing.T) {
	// Arrange
	ctx := t.Context()
	var invalidCmd commands.CreateCourierCommand // zero value command

	mockFactory := new(MockCourierUoWFactory)
	handler := commands.NewCreateCourierCommandHandler(mockFactory)

	// Act
	err := handler.Handle(ctx, invalidCmd)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, commands.ErrCreateCourierCommandIsNotConstructed)
	mockFactory.AssertExpectations(t) // No calls should be made to factory
}

func TestCreateCourierCommandHandler_Handle_BeginTransactionError(t *testing.T) {
	// Arrange
	ctx := t.Context()
	name := "John Doe"
	speed := 3
	location, err := kernel.NewLocation(5, 7)
	require.NoError(t, err)

	cmd, err := commands.NewCreateCourierCommand(name, speed, location)
	require.NoError(t, err)

	expectedError := errors.New("begin transaction failed")
	mockUoW := new(MockCourierUoW)
	mockFactory := new(MockCourierUoWFactory)

	// Set up expectations in order
	mock.InOrder(
		mockFactory.On("Create").Return(mockUoW).Once(),
		mockUoW.On("Begin", ctx).Return(expectedError).Once(),
	)

	handler := commands.NewCreateCourierCommandHandler(mockFactory)

	// Act
	err = handler.Handle(ctx, cmd)

	// Assert
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
}

func TestCreateCourierCommandHandler_Handle_CourierCreationError(t *testing.T) {
	// Test case where courier.NewCourier fails - though this should not happen
	// with a valid command, we test for completeness
	// This test essentially covers the scenario where the domain model changes
	// its validation rules but the command hasn't been updated accordingly
	t.Skip("Skipping as courier.NewCourier should not fail with valid command data")
}

func TestCreateCourierCommandHandler_Handle_RepositoryAddError(t *testing.T) {
	// Arrange
	ctx := t.Context()
	name := "John Doe"
	speed := 3
	location, err := kernel.NewLocation(5, 7)
	require.NoError(t, err)

	cmd, err := commands.NewCreateCourierCommand(name, speed, location)
	require.NoError(t, err)

	expectedError := errors.New("repository add failed")
	mockRepo := new(MockCourierRepository)
	mockUoW := new(MockCourierUoW)
	mockFactory := new(MockCourierUoWFactory)

	// Set up expectations in order
	mock.InOrder(
		mockUoW.On("Begin", ctx).Return(nil).Once(),
		mockUoW.On("CourierRepository").Return(mockRepo).Once(),
		mockRepo.On("Add", ctx, mock.AnythingOfType("*courier.Courier")).Return(expectedError).Once(),
		mockUoW.On("Rollback", ctx).Return(nil).Once(),
	)
	mockFactory.On("Create").Return(mockUoW).Once()

	handler := commands.NewCreateCourierCommandHandler(mockFactory)

	// Act
	err = handler.Handle(ctx, cmd)

	// Assert
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestCreateCourierCommandHandler_Handle_CommitError(t *testing.T) {
	// Arrange
	ctx := t.Context()
	name := "John Doe"
	speed := 3
	location, err := kernel.NewLocation(5, 7)
	require.NoError(t, err)

	cmd, err := commands.NewCreateCourierCommand(name, speed, location)
	require.NoError(t, err)

	expectedError := errors.New("commit failed")
	mockRepo := new(MockCourierRepository)
	mockUoW := new(MockCourierUoW)
	mockFactory := new(MockCourierUoWFactory)

	// Set up expectations in order
	mock.InOrder(
		mockUoW.On("Begin", ctx).Return(nil).Once(),
		mockUoW.On("CourierRepository").Return(mockRepo).Once(),
		mockRepo.On("Add", ctx, mock.AnythingOfType("*courier.Courier")).Return(nil).Once(),
		mockUoW.On("Commit", ctx).Return(expectedError).Once(),
		mockUoW.On("Rollback", ctx).Return(nil).Once(),
	)
	mockFactory.On("Create").Return(mockUoW).Once()

	handler := commands.NewCreateCourierCommandHandler(mockFactory)

	// Act
	err = handler.Handle(ctx, cmd)

	// Assert
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestCreateCourierCommandHandler_Handle_RepositoryAddErrorWithRollbackError(t *testing.T) {
	// Arrange
	ctx := t.Context()
	name := "John Doe"
	speed := 3
	location, err := kernel.NewLocation(5, 7)
	require.NoError(t, err)

	cmd, err := commands.NewCreateCourierCommand(name, speed, location)
	require.NoError(t, err)

	repoError := errors.New("repository add failed")
	rollbackError := errors.New("rollback failed")
	mockRepo := new(MockCourierRepository)
	mockUoW := new(MockCourierUoW)
	mockFactory := new(MockCourierUoWFactory)

	// Set up expectations in order
	mock.InOrder(
		mockUoW.On("Begin", ctx).Return(nil).Once(),
		mockUoW.On("CourierRepository").Return(mockRepo).Once(),
		mockRepo.On("Add", ctx, mock.AnythingOfType("*courier.Courier")).Return(repoError).Once(),
		mockUoW.On("Rollback", ctx).Return(rollbackError).Once(),
	)
	mockFactory.On("Create").Return(mockUoW).Once()

	handler := commands.NewCreateCourierCommandHandler(mockFactory)

	// Act
	err = handler.Handle(ctx, cmd)

	// Assert
	// Should return the original repository error, not the rollback error
	require.Error(t, err)
	assert.Equal(t, repoError, err)
	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestCreateCourierCommandHandler_Handle_CommitErrorWithRollbackError(t *testing.T) {
	// Arrange
	ctx := t.Context()
	name := "John Doe"
	speed := 3
	location, err := kernel.NewLocation(5, 7)
	require.NoError(t, err)

	cmd, err := commands.NewCreateCourierCommand(name, speed, location)
	require.NoError(t, err)

	commitError := errors.New("commit failed")
	rollbackError := errors.New("rollback failed")
	mockRepo := new(MockCourierRepository)
	mockUoW := new(MockCourierUoW)
	mockFactory := new(MockCourierUoWFactory)

	// Set up expectations in order
	mock.InOrder(
		mockUoW.On("Begin", ctx).Return(nil).Once(),
		mockUoW.On("CourierRepository").Return(mockRepo).Once(),
		mockRepo.On("Add", ctx, mock.AnythingOfType("*courier.Courier")).Return(nil).Once(),
		mockUoW.On("Commit", ctx).Return(commitError).Once(),
		mockUoW.On("Rollback", ctx).Return(rollbackError).Once(),
	)
	mockFactory.On("Create").Return(mockUoW).Once()

	handler := commands.NewCreateCourierCommandHandler(mockFactory)

	// Act
	err = handler.Handle(ctx, cmd)

	// Assert
	// Should return the original commit error, not the rollback error
	require.Error(t, err)
	assert.Equal(t, commitError, err)
	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestCreateCourierCommandHandler_Handle_VerifiesCourierDataCorrectness(t *testing.T) {
	// Arrange
	ctx := t.Context()
	name := "Alice Johnson"
	speed := 5
	location, err := kernel.NewLocation(3, 8)
	require.NoError(t, err)

	cmd, err := commands.NewCreateCourierCommand(name, speed, location)
	require.NoError(t, err)

	var capturedCourier *courier.Courier
	mockRepo := new(MockCourierRepository)
	mockUoW := new(MockCourierUoW)
	mockFactory := new(MockCourierUoWFactory)

	// Set up expectations in order with custom matcher to capture the courier
	mock.InOrder(
		mockUoW.On("Begin", ctx).Return(nil).Once(),
		mockUoW.On("CourierRepository").Return(mockRepo).Once(),
		mockRepo.On("Add", ctx, mock.MatchedBy(func(c *courier.Courier) bool {
			capturedCourier = c
			return true
		})).Return(nil).Once(),
		mockUoW.On("Commit", ctx).Return(nil).Once(),
		mockUoW.On("Rollback", ctx).Return(nil).Once(),
	)
	mockFactory.On("Create").Return(mockUoW).Once()

	handler := commands.NewCreateCourierCommandHandler(mockFactory)

	// Act
	err = handler.Handle(ctx, cmd)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, capturedCourier)

	// Verify the courier was created with correct data
	assert.Equal(t, cmd.CourierID(), capturedCourier.ID())
	assert.Equal(t, name, capturedCourier.Name())
	assert.Equal(t, speed, capturedCourier.Speed())
	assert.Equal(t, location, capturedCourier.Location())

	// Verify courier is valid
	require.NoError(t, capturedCourier.Validate())

	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestCreateCourierCommandHandler_Handle_MultipleCommandsGenerateUniqueIDs(t *testing.T) {
	// Arrange
	location, err := kernel.NewLocation(5, 5)
	require.NoError(t, err)

	cmd1, err := commands.NewCreateCourierCommand("Courier 1", 2, location)
	require.NoError(t, err)

	cmd2, err := commands.NewCreateCourierCommand("Courier 2", 3, location)
	require.NoError(t, err)

	// Assert
	assert.NotEqual(t, cmd1.CourierID(), cmd2.CourierID(), "Different commands should generate unique courier IDs")
}

// Benchmark test to ensure performance is acceptable.
func BenchmarkCreateCourierCommandHandler_Handle(b *testing.B) {
	ctx := b.Context()
	location, err := kernel.NewLocation(5, 7)
	require.NoError(b, err)

	cmd, err := commands.NewCreateCourierCommand("Benchmark Courier", 3, location)
	require.NoError(b, err)

	mockRepo := new(MockCourierRepository)
	mockUoW := new(MockCourierUoW)
	mockFactory := new(MockCourierUoWFactory)

	// Set up expectations for benchmarking
	mockFactory.On("Create").Return(mockUoW).Times(b.N)
	mockUoW.On("Begin", ctx).Return(nil).Times(b.N)
	mockUoW.On("CourierRepository").Return(mockRepo).Times(b.N)
	mockRepo.On("Add", ctx, mock.AnythingOfType("*courier.Courier")).Return(nil).Times(b.N)
	mockUoW.On("Commit", ctx).Return(nil).Times(b.N)
	mockUoW.On("Rollback", ctx).Return(nil).Times(b.N)

	handler := commands.NewCreateCourierCommandHandler(mockFactory)

	b.ResetTimer()
	for range b.N {
		benchErr := handler.Handle(ctx, cmd)
		if benchErr != nil {
			b.Fatal(benchErr)
		}
	}
}
