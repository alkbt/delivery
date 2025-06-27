package commands_test

import (
	"testing"

	"delivery/internal/core/application/usecases/commands"
	"delivery/internal/core/domain/model/kernel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAddCourierStorageCommand_ValidInput(t *testing.T) {
	// Arrange
	courierID := kernel.NewUUID()
	name := "Thermal Bag"
	totalVolume := 50

	// Act
	cmd, err := commands.NewAddCourierStorageCommand(courierID, name, totalVolume)

	// Assert
	require.NoError(t, err)
	assert.NotZero(t, cmd)
	assert.Equal(t, courierID, cmd.CourierID())
	assert.Equal(t, name, cmd.Name())
	assert.Equal(t, totalVolume, cmd.TotalVolume())
	assert.NoError(t, cmd.Validate())
}

func TestNewAddCourierStorageCommand_ValidInputBoundaryValues(t *testing.T) {
	testCases := []struct {
		name        string
		storageName string
		totalVolume int
	}{
		{
			name:        "minimum volume",
			storageName: "Small Bag",
			totalVolume: 1,
		},
		{
			name:        "large volume",
			storageName: "Large Container",
			totalVolume: 1000,
		},
		{
			name:        "single character name",
			storageName: "X",
			totalVolume: 10,
		},
		{
			name:        "long storage name",
			storageName: "Very Long Storage Container Name With Many Characters For Testing",
			totalVolume: 100,
		},
		{
			name:        "storage with special characters",
			storageName: "Thermal Bag #1 (Large)",
			totalVolume: 75,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			courierID := kernel.NewUUID()

			// Act
			cmd, err := commands.NewAddCourierStorageCommand(courierID, tc.storageName, tc.totalVolume)

			// Assert
			require.NoError(t, err)
			assert.NotZero(t, cmd)
			assert.Equal(t, courierID, cmd.CourierID())
			assert.Equal(t, tc.storageName, cmd.Name())
			assert.Equal(t, tc.totalVolume, cmd.TotalVolume())
			assert.NoError(t, cmd.Validate())
		})
	}
}

func TestNewAddCourierStorageCommand_InvalidCourierID(t *testing.T) {
	// Arrange
	var invalidCourierID kernel.UUID // zero value
	name := "Storage"
	totalVolume := 50

	// Act
	_, err := commands.NewAddCourierStorageCommand(invalidCourierID, name, totalVolume)

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, kernel.ErrUUIDIsNotConstructed)
}

func TestNewAddCourierStorageCommand_EmptyName(t *testing.T) {
	// Arrange
	courierID := kernel.NewUUID()
	totalVolume := 50

	// Act
	_, err := commands.NewAddCourierStorageCommand(courierID, "", totalVolume)

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, commands.ErrNameIsRequired)
}

func TestNewAddCourierStorageCommand_InvalidTotalVolume(t *testing.T) {
	testCases := []struct {
		name        string
		totalVolume int
	}{
		{
			name:        "zero volume",
			totalVolume: 0,
		},
		{
			name:        "negative volume",
			totalVolume: -1,
		},
		{
			name:        "very negative volume",
			totalVolume: -100,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			courierID := kernel.NewUUID()
			name := "Storage"

			// Act
			_, err := commands.NewAddCourierStorageCommand(courierID, name, tc.totalVolume)

			// Assert
			require.Error(t, err)
			assert.ErrorIs(t, err, commands.ErrTotalVolumeIsInvalid)
		})
	}
}

func TestNewAddCourierStorageCommand_MultipleCombinedErrors(t *testing.T) {
	// Arrange
	var invalidCourierID kernel.UUID // zero value

	// Act
	_, err := commands.NewAddCourierStorageCommand(invalidCourierID, "", 0)

	// Assert
	require.Error(t, err)
	// Should contain multiple errors - courier ID, name and total volume validation failures
	assert.Contains(t, err.Error(), "UUID must be created via NewUUID, UUIDFromString, or UUIDFromBytes")
	assert.Contains(t, err.Error(), "name is required")
	assert.Contains(t, err.Error(), "total volume must be greater than 0")
}

func TestNewAddCourierStorageCommand_PartialErrors(t *testing.T) {
	testCases := []struct {
		name         string
		courierID    kernel.UUID
		storageName  string
		totalVolume  int
		expectedErrs []string
	}{
		{
			name:        "invalid courier ID and empty name",
			courierID:   kernel.UUID{}, // zero value
			storageName: "",
			totalVolume: 50,
			expectedErrs: []string{
				"UUID must be created via NewUUID, UUIDFromString, or UUIDFromBytes",
				"name is required",
			},
		},
		{
			name:        "invalid courier ID and invalid volume",
			courierID:   kernel.UUID{}, // zero value
			storageName: "Storage",
			totalVolume: -5,
			expectedErrs: []string{
				"UUID must be created via NewUUID, UUIDFromString, or UUIDFromBytes",
				"total volume must be greater than 0",
			},
		},
		{
			name:        "empty name and invalid volume",
			courierID:   kernel.NewUUID(),
			storageName: "",
			totalVolume: 0,
			expectedErrs: []string{
				"name is required",
				"total volume must be greater than 0",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			_, err := commands.NewAddCourierStorageCommand(tc.courierID, tc.storageName, tc.totalVolume)

			// Assert
			require.Error(t, err)
			for _, expectedErr := range tc.expectedErrs {
				assert.Contains(t, err.Error(), expectedErr)
			}
		})
	}
}

func TestAddCourierStorageCommand_Validate_Success(t *testing.T) {
	// Arrange
	courierID := kernel.NewUUID()
	cmd, err := commands.NewAddCourierStorageCommand(courierID, "Storage", 50)
	require.NoError(t, err)

	// Act
	err = cmd.Validate()

	// Assert
	assert.NoError(t, err)
}

func TestAddCourierStorageCommand_Validate_ZeroValue(t *testing.T) {
	// Arrange
	var cmd commands.AddCourierStorageCommand // zero value

	// Act
	err := cmd.Validate()

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, commands.ErrAddCourierStorageCommandIsNotConstructed)
}

func TestAddCourierStorageCommand_Validate_ErrorType(t *testing.T) {
	// Arrange
	var cmd commands.AddCourierStorageCommand // zero value

	// Act
	err := cmd.Validate()

	// Assert
	require.Error(t, err)

	// Test that ErrAddCourierStorageCommandIsNotConstructed is the specific error returned
	expectedErr := commands.ErrAddCourierStorageCommandIsNotConstructed
	assert.Equal(
		t,
		"AddCourierStorageCommand must be created via NewAddCourierStorageCommand constructor",
		expectedErr.Error(),
	)
	assert.ErrorIs(t, err, expectedErr)
}

func TestAddCourierStorageCommand_GetterMethods(t *testing.T) {
	// Arrange
	courierID := kernel.NewUUID()
	name := "Premium Storage"
	totalVolume := 150

	cmd, err := commands.NewAddCourierStorageCommand(courierID, name, totalVolume)
	require.NoError(t, err)

	// Act & Assert
	assert.Equal(t, courierID, cmd.CourierID())
	assert.Equal(t, name, cmd.Name())
	assert.Equal(t, totalVolume, cmd.TotalVolume())
}

func TestAddCourierStorageCommand_GetterMethods_ZeroValueReturnsDefaults(t *testing.T) {
	// Arrange
	var cmd commands.AddCourierStorageCommand // zero value

	// Act & Assert
	assert.Zero(t, cmd.CourierID())
	assert.Empty(t, cmd.Name())
	assert.Zero(t, cmd.TotalVolume())
}

func TestAddCourierStorageCommand_ImmutabilityAfterCreation(t *testing.T) {
	// Arrange
	courierID := kernel.NewUUID()
	originalName := "Original Storage"
	originalVolume := 100

	cmd, err := commands.NewAddCourierStorageCommand(courierID, originalName, originalVolume)
	require.NoError(t, err)

	// Act - Get values multiple times
	name1 := cmd.Name()
	name2 := cmd.Name()
	volume1 := cmd.TotalVolume()
	volume2 := cmd.TotalVolume()
	id1 := cmd.CourierID()
	id2 := cmd.CourierID()

	// Assert - Values should remain consistent
	assert.Equal(t, name1, name2)
	assert.Equal(t, volume1, volume2)
	assert.True(t, id1.IsEqual(id2))
	assert.Equal(t, originalName, name1)
	assert.Equal(t, originalVolume, volume1)
	assert.True(t, courierID.IsEqual(id1))
}

func TestAddCourierStorageCommand_ErrorConstants(t *testing.T) {
	// Test that error constants have expected messages
	testCases := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{
			name:        "ErrAddCourierStorageCommandIsNotConstructed",
			err:         commands.ErrAddCourierStorageCommandIsNotConstructed,
			expectedMsg: "AddCourierStorageCommand must be created via NewAddCourierStorageCommand constructor",
		},
		{
			name:        "ErrTotalVolumeIsInvalid",
			err:         commands.ErrTotalVolumeIsInvalid,
			expectedMsg: "total volume must be greater than 0",
		},
		{
			name:        "ErrNameIsRequired",
			err:         commands.ErrNameIsRequired,
			expectedMsg: "name is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedMsg, tc.err.Error())
		})
	}
}

func BenchmarkNewAddCourierStorageCommand(b *testing.B) {
	courierID := kernel.NewUUID()
	name := "Storage"
	totalVolume := 50

	b.ResetTimer()
	for range b.N {
		_, _ = commands.NewAddCourierStorageCommand(courierID, name, totalVolume)
	}
}

func BenchmarkAddCourierStorageCommand_Validate(b *testing.B) {
	courierID := kernel.NewUUID()
	cmd, err := commands.NewAddCourierStorageCommand(courierID, "Storage", 50)
	require.NoError(b, err)

	b.ResetTimer()
	for range b.N {
		_ = cmd.Validate()
	}
}
