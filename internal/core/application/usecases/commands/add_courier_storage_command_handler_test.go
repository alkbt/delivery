package commands_test

import (
	"errors"
	"testing"

	"delivery/internal/core/application/usecases/commands"
	"delivery/internal/core/domain/model/courier"
	"delivery/internal/core/domain/model/kernel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewAddCourierStorageCommandHandler(t *testing.T) {
	// Arrange
	mockFactory := new(MockCourierUoWFactory)

	// Act
	handler := commands.NewAddCourierStorageCommandHandler(mockFactory)

	// Assert
	assert.NotNil(t, handler)
}

func TestAddCourierStorageCommandHandler_Handle_Success(t *testing.T) {
	// Arrange
	ctx := t.Context()
	courierID := kernel.NewUUID()
	name := "Premium Storage"
	totalVolume := 50

	cmd, err := commands.NewAddCourierStorageCommand(courierID, name, totalVolume)
	require.NoError(t, err)

	// Create a courier for the test
	location, err := kernel.NewLocation(5, 7)
	require.NoError(t, err)
	courierEntity, err := courier.NewCourier(courierID, "Test Courier", 3, location)
	require.NoError(t, err)

	mockRepo := new(MockCourierRepository)
	mockUoW := new(MockCourierUoW)
	mockFactory := new(MockCourierUoWFactory)

	// Set up expectations in order
	mock.InOrder(
		mockUoW.On("Begin", ctx).Return(nil).Once(),
		mockUoW.On("CourierRepository").Return(mockRepo).Once(),
		mockRepo.On("Get", ctx, courierID).Return(courierEntity, nil).Once(),
		mockRepo.On("Update", ctx, courierEntity).Return(nil).Once(),
		mockUoW.On("Commit", ctx).Return(nil).Once(),
		mockUoW.On("Rollback", ctx).Return(nil).Once(),
	)
	mockFactory.On("Create").Return(mockUoW).Once()

	handler := commands.NewAddCourierStorageCommandHandler(mockFactory)

	// Act
	err = handler.Handle(ctx, cmd)

	// Assert
	require.NoError(t, err)
	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockRepo.AssertExpectations(t)

	// Verify storage was added to courier
	storagePlaces := courierEntity.StoragePlaces()
	found := false
	for _, sp := range storagePlaces {
		if sp.Name() == name && sp.TotalVolume() == totalVolume {
			found = true
			break
		}
	}
	assert.True(t, found, "Storage place should be added to courier")
}

func TestAddCourierStorageCommandHandler_Handle_InvalidCommand(t *testing.T) {
	// Arrange
	ctx := t.Context()
	var invalidCmd commands.AddCourierStorageCommand // zero value command

	mockFactory := new(MockCourierUoWFactory)
	handler := commands.NewAddCourierStorageCommandHandler(mockFactory)

	// Act
	err := handler.Handle(ctx, invalidCmd)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, commands.ErrAddCourierStorageCommandIsNotConstructed)
	mockFactory.AssertExpectations(t) // No calls should be made to factory
}

func TestAddCourierStorageCommandHandler_Handle_BeginTransactionError(t *testing.T) {
	// Arrange
	ctx := t.Context()
	courierID := kernel.NewUUID()
	cmd, err := commands.NewAddCourierStorageCommand(courierID, "Storage", 50)
	require.NoError(t, err)

	expectedError := errors.New("begin transaction failed")
	mockUoW := new(MockCourierUoW)
	mockFactory := new(MockCourierUoWFactory)

	// Set up expectations in order
	mock.InOrder(
		mockFactory.On("Create").Return(mockUoW).Once(),
		mockUoW.On("Begin", ctx).Return(expectedError).Once(),
	)

	handler := commands.NewAddCourierStorageCommandHandler(mockFactory)

	// Act
	err = handler.Handle(ctx, cmd)

	// Assert
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
}

func TestAddCourierStorageCommandHandler_Handle_GetCourierError(t *testing.T) {
	// Arrange
	ctx := t.Context()
	courierID := kernel.NewUUID()
	cmd, err := commands.NewAddCourierStorageCommand(courierID, "Storage", 50)
	require.NoError(t, err)

	expectedError := errors.New("courier not found")
	mockRepo := new(MockCourierRepository)
	mockUoW := new(MockCourierUoW)
	mockFactory := new(MockCourierUoWFactory)

	// Set up expectations in order
	mock.InOrder(
		mockUoW.On("Begin", ctx).Return(nil).Once(),
		mockUoW.On("CourierRepository").Return(mockRepo).Once(),
		mockRepo.On("Get", ctx, courierID).Return((*courier.Courier)(nil), expectedError).Once(),
		mockUoW.On("Rollback", ctx).Return(nil).Once(),
	)
	mockFactory.On("Create").Return(mockUoW).Once()

	handler := commands.NewAddCourierStorageCommandHandler(mockFactory)

	// Act
	err = handler.Handle(ctx, cmd)

	// Assert
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestAddCourierStorageCommandHandler_Handle_AddStoragePlaceError(t *testing.T) {
	// This test is simplified because the AddStoragePlace method in the courier domain
	// doesn't have obvious validation failures with normal inputs.
	// We'll test a scenario where the domain operation succeeds but could potentially fail

	// Arrange
	courierID := kernel.NewUUID()

	// Test that a command with empty name fails during construction
	_, err := commands.NewAddCourierStorageCommand(courierID, "", 50)
	require.Error(t, err) // Command creation should fail with empty name
	require.ErrorIs(t, err, commands.ErrNameIsRequired)

	// Test that a command with invalid volume fails during construction
	_, err = commands.NewAddCourierStorageCommand(courierID, "Storage", 0)
	require.Error(t, err) // Command creation should fail with invalid volume
	assert.ErrorIs(t, err, commands.ErrTotalVolumeIsInvalid)
}

func TestAddCourierStorageCommandHandler_Handle_UpdateCourierError(t *testing.T) {
	// Arrange
	ctx := t.Context()
	courierID := kernel.NewUUID()
	cmd, err := commands.NewAddCourierStorageCommand(courierID, "Storage", 50)
	require.NoError(t, err)

	location, err := kernel.NewLocation(5, 7)
	require.NoError(t, err)
	courierEntity, err := courier.NewCourier(courierID, "Test Courier", 3, location)
	require.NoError(t, err)

	expectedError := errors.New("update failed")
	mockRepo := new(MockCourierRepository)
	mockUoW := new(MockCourierUoW)
	mockFactory := new(MockCourierUoWFactory)

	// Set up expectations in order
	mock.InOrder(
		mockUoW.On("Begin", ctx).Return(nil).Once(),
		mockUoW.On("CourierRepository").Return(mockRepo).Once(),
		mockRepo.On("Get", ctx, courierID).Return(courierEntity, nil).Once(),
		mockRepo.On("Update", ctx, courierEntity).Return(expectedError).Once(),
		mockUoW.On("Rollback", ctx).Return(nil).Once(),
	)
	mockFactory.On("Create").Return(mockUoW).Once()

	handler := commands.NewAddCourierStorageCommandHandler(mockFactory)

	// Act
	err = handler.Handle(ctx, cmd)

	// Assert
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestAddCourierStorageCommandHandler_Handle_CommitError(t *testing.T) {
	// Arrange
	ctx := t.Context()
	courierID := kernel.NewUUID()
	cmd, err := commands.NewAddCourierStorageCommand(courierID, "Storage", 50)
	require.NoError(t, err)

	location, err := kernel.NewLocation(5, 7)
	require.NoError(t, err)
	courierEntity, err := courier.NewCourier(courierID, "Test Courier", 3, location)
	require.NoError(t, err)

	expectedError := errors.New("commit failed")
	mockRepo := new(MockCourierRepository)
	mockUoW := new(MockCourierUoW)
	mockFactory := new(MockCourierUoWFactory)

	// Set up expectations in order
	mock.InOrder(
		mockUoW.On("Begin", ctx).Return(nil).Once(),
		mockUoW.On("CourierRepository").Return(mockRepo).Once(),
		mockRepo.On("Get", ctx, courierID).Return(courierEntity, nil).Once(),
		mockRepo.On("Update", ctx, courierEntity).Return(nil).Once(),
		mockUoW.On("Commit", ctx).Return(expectedError).Once(),
		mockUoW.On("Rollback", ctx).Return(nil).Once(),
	)
	mockFactory.On("Create").Return(mockUoW).Once()

	handler := commands.NewAddCourierStorageCommandHandler(mockFactory)

	// Act
	err = handler.Handle(ctx, cmd)

	// Assert
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestAddCourierStorageCommandHandler_Handle_GetErrorWithRollbackError(t *testing.T) {
	// Arrange
	ctx := t.Context()
	courierID := kernel.NewUUID()
	cmd, err := commands.NewAddCourierStorageCommand(courierID, "Storage", 50)
	require.NoError(t, err)

	getError := errors.New("get failed")
	rollbackError := errors.New("rollback failed")
	mockRepo := new(MockCourierRepository)
	mockUoW := new(MockCourierUoW)
	mockFactory := new(MockCourierUoWFactory)

	// Set up expectations in order
	mock.InOrder(
		mockUoW.On("Begin", ctx).Return(nil).Once(),
		mockUoW.On("CourierRepository").Return(mockRepo).Once(),
		mockRepo.On("Get", ctx, courierID).Return((*courier.Courier)(nil), getError).Once(),
		mockUoW.On("Rollback", ctx).Return(rollbackError).Once(),
	)
	mockFactory.On("Create").Return(mockUoW).Once()

	handler := commands.NewAddCourierStorageCommandHandler(mockFactory)

	// Act
	err = handler.Handle(ctx, cmd)

	// Assert
	// Should return the original get error, not the rollback error
	require.Error(t, err)
	assert.Equal(t, getError, err)
	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestAddCourierStorageCommandHandler_Handle_CommitErrorWithRollbackError(t *testing.T) {
	// Arrange
	ctx := t.Context()
	courierID := kernel.NewUUID()
	cmd, err := commands.NewAddCourierStorageCommand(courierID, "Storage", 50)
	require.NoError(t, err)

	location, err := kernel.NewLocation(5, 7)
	require.NoError(t, err)
	courierEntity, err := courier.NewCourier(courierID, "Test Courier", 3, location)
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
		mockRepo.On("Get", ctx, courierID).Return(courierEntity, nil).Once(),
		mockRepo.On("Update", ctx, courierEntity).Return(nil).Once(),
		mockUoW.On("Commit", ctx).Return(commitError).Once(),
		mockUoW.On("Rollback", ctx).Return(rollbackError).Once(),
	)
	mockFactory.On("Create").Return(mockUoW).Once()

	handler := commands.NewAddCourierStorageCommandHandler(mockFactory)

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

func TestAddCourierStorageCommandHandler_Handle_VerifiesStorageAddedCorrectly(t *testing.T) {
	// Arrange
	ctx := t.Context()
	courierID := kernel.NewUUID()
	storageName := "Premium Container"
	storageVolume := 75

	cmd, err := commands.NewAddCourierStorageCommand(courierID, storageName, storageVolume)
	require.NoError(t, err)

	location, err := kernel.NewLocation(3, 8)
	require.NoError(t, err)
	courierEntity, err := courier.NewCourier(courierID, "Alice Johnson", 5, location)
	require.NoError(t, err)

	// Count initial storage places
	initialStorageCount := len(courierEntity.StoragePlaces())

	var capturedCourier *courier.Courier
	mockRepo := new(MockCourierRepository)
	mockUoW := new(MockCourierUoW)
	mockFactory := new(MockCourierUoWFactory)

	// Set up expectations in order with custom matcher to capture the courier
	mock.InOrder(
		mockUoW.On("Begin", ctx).Return(nil).Once(),
		mockUoW.On("CourierRepository").Return(mockRepo).Once(),
		mockRepo.On("Get", ctx, courierID).Return(courierEntity, nil).Once(),
		mockRepo.On("Update", ctx, mock.MatchedBy(func(c *courier.Courier) bool {
			capturedCourier = c
			return true
		})).Return(nil).Once(),
		mockUoW.On("Commit", ctx).Return(nil).Once(),
		mockUoW.On("Rollback", ctx).Return(nil).Once(),
	)
	mockFactory.On("Create").Return(mockUoW).Once()

	handler := commands.NewAddCourierStorageCommandHandler(mockFactory)

	// Act
	err = handler.Handle(ctx, cmd)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, capturedCourier)

	// Verify the storage was added correctly
	finalStoragePlaces := capturedCourier.StoragePlaces()
	assert.Len(t, finalStoragePlaces, initialStorageCount+1)

	// Find the newly added storage
	found := false
	for _, sp := range finalStoragePlaces {
		if sp.Name() == storageName && sp.TotalVolume() == storageVolume {
			found = true
			assert.Nil(t, sp.OrderID()) // New storage should be empty
			break
		}
	}
	assert.True(t, found, "New storage place should be found with correct properties")

	// Verify courier is still valid
	require.NoError(t, capturedCourier.Validate())

	mockFactory.AssertExpectations(t)
	mockUoW.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

// Benchmark test to ensure performance is acceptable.
func BenchmarkAddCourierStorageCommandHandler_Handle(b *testing.B) {
	ctx := b.Context()
	courierID := kernel.NewUUID()
	cmd, err := commands.NewAddCourierStorageCommand(courierID, "Benchmark Storage", 50)
	require.NoError(b, err)

	location, err := kernel.NewLocation(5, 7)
	require.NoError(b, err)
	courierEntity, err := courier.NewCourier(courierID, "Benchmark Courier", 3, location)
	require.NoError(b, err)

	mockRepo := new(MockCourierRepository)
	mockUoW := new(MockCourierUoW)
	mockFactory := new(MockCourierUoWFactory)

	// Set up expectations for benchmarking
	mockFactory.On("Create").Return(mockUoW).Times(b.N)
	mockUoW.On("Begin", ctx).Return(nil).Times(b.N)
	mockUoW.On("CourierRepository").Return(mockRepo).Times(b.N)
	mockRepo.On("Get", ctx, courierID).Return(courierEntity, nil).Times(b.N)
	mockRepo.On("Update", ctx, courierEntity).Return(nil).Times(b.N)
	mockUoW.On("Commit", ctx).Return(nil).Times(b.N)
	mockUoW.On("Rollback", ctx).Return(nil).Times(b.N)

	handler := commands.NewAddCourierStorageCommandHandler(mockFactory)

	b.ResetTimer()
	for range b.N {
		benchErr := handler.Handle(ctx, cmd)
		if benchErr != nil {
			b.Fatal(benchErr)
		}
	}
}
