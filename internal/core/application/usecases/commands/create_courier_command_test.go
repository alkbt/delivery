package commands_test

import (
	"testing"

	"delivery/internal/core/application/usecases/commands"
	"delivery/internal/core/domain/model/kernel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCreateCourierCommand_ValidInput(t *testing.T) {
	// Arrange
	name := "John Doe"
	speed := 3
	location, err := kernel.NewLocation(5, 7)
	require.NoError(t, err)

	// Act
	cmd, err := commands.NewCreateCourierCommand(name, speed, location)

	// Assert
	require.NoError(t, err)
	assert.NotZero(t, cmd)
	assert.Equal(t, name, cmd.Name())
	assert.Equal(t, speed, cmd.Speed())
	assert.Equal(t, location, cmd.Location())
	assert.NotZero(t, cmd.CourierID())

	// Verify the courier ID is valid
	assert.NoError(t, cmd.CourierID().Validate())
}

func TestNewCreateCourierCommand_ValidInputBoundaryValues(t *testing.T) {
	// Test with boundary location values
	testCases := []struct {
		name        string
		courierName string
		speed       int
		x, y        kernel.Coordinate
	}{
		{
			name:        "min coordinates",
			courierName: "Courier A",
			speed:       1,
			x:           kernel.LocationMinX,
			y:           kernel.LocationMinY,
		},
		{
			name:        "max coordinates",
			courierName: "Courier B",
			speed:       5,
			x:           kernel.LocationMaxX,
			y:           kernel.LocationMaxY,
		},
		{
			name:        "single character name",
			courierName: "X",
			speed:       2,
			x:           5,
			y:           5,
		},
		{
			name:        "long name",
			courierName: "Very Long Courier Name With Many Characters",
			speed:       10,
			x:           3,
			y:           8,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			location, err := kernel.NewLocation(tc.x, tc.y)
			require.NoError(t, err)

			// Act
			cmd, err := commands.NewCreateCourierCommand(tc.courierName, tc.speed, location)

			// Assert
			require.NoError(t, err)
			assert.NotZero(t, cmd)
			assert.Equal(t, tc.courierName, cmd.Name())
			assert.Equal(t, tc.speed, cmd.Speed())
			assert.Equal(t, location, cmd.Location())
			assert.NotZero(t, cmd.CourierID())
			assert.NoError(t, cmd.CourierID().Validate())
		})
	}
}

func TestNewCreateCourierCommand_EmptyName(t *testing.T) {
	// Arrange
	speed := 3
	location, err := kernel.NewLocation(5, 7)
	require.NoError(t, err)

	// Act
	_, err = commands.NewCreateCourierCommand("", speed, location)

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, commands.ErrNameIsRequired)
}

func TestNewCreateCourierCommand_InvalidSpeed(t *testing.T) {
	testCases := []struct {
		name  string
		speed int
	}{
		{
			name:  "zero speed",
			speed: 0,
		},
		{
			name:  "negative speed",
			speed: -1,
		},
		{
			name:  "very negative speed",
			speed: -100,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			name := "John Doe"
			location, err := kernel.NewLocation(5, 7)
			require.NoError(t, err)

			// Act
			_, err = commands.NewCreateCourierCommand(name, tc.speed, location)

			// Assert
			require.Error(t, err)
			assert.ErrorIs(t, err, commands.ErrSpeedIsInvalid)
		})
	}
}

func TestNewCreateCourierCommand_InvalidLocation_OutOfRangeX(t *testing.T) {
	// Arrange
	name := "John Doe"
	speed := 3

	// Act
	_, err := commands.NewCreateCourierCommand(name, speed, kernel.Location{})

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, kernel.ErrLocationIsNotConstructed)
}

func TestNewCreateCourierCommand_InvalidLocation_ZeroValue(t *testing.T) {
	// Arrange
	name := "John Doe"
	speed := 3
	var invalidLocation kernel.Location // zero value

	// Act
	_, err := commands.NewCreateCourierCommand(name, speed, invalidLocation)

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, kernel.ErrLocationIsNotConstructed)
}

func TestNewCreateCourierCommand_MultipleCombinedErrors(t *testing.T) {
	// Arrange
	var invalidLocation kernel.Location // zero value

	// Act
	_, err := commands.NewCreateCourierCommand("", 0, invalidLocation)

	// Assert
	require.Error(t, err)
	// Should contain multiple errors - name, speed and location validation failures
	assert.Contains(t, err.Error(), "name is required")
	assert.Contains(t, err.Error(), "speed must be greater than 0")
	assert.Contains(t, err.Error(), "location must be created via NewLocation or NewRandomLocation constructors")
}

func TestNewCreateCourierCommand_WithRandomLocation(t *testing.T) {
	// Arrange
	name := "Random Courier"
	speed := 4
	location, err := kernel.NewRandomLocation()
	require.NoError(t, err)

	// Act
	cmd, err := commands.NewCreateCourierCommand(name, speed, location)

	// Assert
	require.NoError(t, err)
	assert.NotZero(t, cmd)
	assert.Equal(t, name, cmd.Name())
	assert.Equal(t, speed, cmd.Speed())
	assert.Equal(t, location, cmd.Location())
	assert.NotZero(t, cmd.CourierID())
	assert.NoError(t, cmd.CourierID().Validate())
}

func TestNewCreateCourierCommand_NameWithSpecialCharacters(t *testing.T) {
	// Test names with various special characters
	testCases := []struct {
		name        string
		courierName string
		shouldPass  bool
	}{
		{
			name:        "name with spaces",
			courierName: "John Doe Smith",
			shouldPass:  true,
		},
		{
			name:        "name with hyphens",
			courierName: "Jean-Pierre",
			shouldPass:  true,
		},
		{
			name:        "name with apostrophe",
			courierName: "O'Connor",
			shouldPass:  true,
		},
		{
			name:        "name with numbers",
			courierName: "Agent007",
			shouldPass:  true,
		},
		{
			name:        "name with unicode",
			courierName: "José María",
			shouldPass:  true,
		},
		{
			name:        "name with special symbols",
			courierName: "@#$%Courier",
			shouldPass:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			speed := 2
			location, err := kernel.NewLocation(5, 5)
			require.NoError(t, err)

			// Act
			cmd, err := commands.NewCreateCourierCommand(tc.courierName, speed, location)

			// Assert
			if tc.shouldPass {
				require.NoError(t, err)
				assert.NotZero(t, cmd)
				assert.Equal(t, tc.courierName, cmd.Name())
				assert.Equal(t, speed, cmd.Speed())
				assert.Equal(t, location, cmd.Location())
				assert.NotZero(t, cmd.CourierID())
				assert.NoError(t, cmd.CourierID().Validate())
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestNewCreateCourierCommand_SpeedBoundaryValues(t *testing.T) {
	testCases := []struct {
		name       string
		speed      int
		shouldPass bool
	}{
		{
			name:       "minimum valid speed",
			speed:      1,
			shouldPass: true,
		},
		{
			name:       "moderate speed",
			speed:      5,
			shouldPass: true,
		},
		{
			name:       "high speed",
			speed:      100,
			shouldPass: true,
		},
		{
			name:       "very high speed",
			speed:      1000,
			shouldPass: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			name := "Speed Test Courier"
			location, err := kernel.NewLocation(5, 5)
			require.NoError(t, err)

			// Act
			cmd, err := commands.NewCreateCourierCommand(name, tc.speed, location)

			// Assert
			if tc.shouldPass {
				require.NoError(t, err)
				assert.NotZero(t, cmd)
				assert.Equal(t, name, cmd.Name())
				assert.Equal(t, tc.speed, cmd.Speed())
				assert.Equal(t, location, cmd.Location())
				assert.NotZero(t, cmd.CourierID())
				assert.NoError(t, cmd.CourierID().Validate())
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestCreateCourierCommand_ErrorVariations(t *testing.T) {
	// Test that ErrNameIsRequired is the specific error returned
	t.Run("name error type verification", func(t *testing.T) {
		location, err := kernel.NewLocation(5, 5)
		require.NoError(t, err)

		_, err = commands.NewCreateCourierCommand("", 3, location)

		require.Error(t, err)
		assert.Equal(t, "name is required", commands.ErrNameIsRequired.Error())
		assert.ErrorIs(t, err, commands.ErrNameIsRequired)
	})

	// Test that ErrSpeedIsInvalid is the specific error returned
	t.Run("speed error type verification", func(t *testing.T) {
		location, err := kernel.NewLocation(5, 5)
		require.NoError(t, err)

		_, err = commands.NewCreateCourierCommand("John Doe", 0, location)

		require.Error(t, err)
		assert.Equal(t, "speed must be greater than 0", commands.ErrSpeedIsInvalid.Error())
		assert.ErrorIs(t, err, commands.ErrSpeedIsInvalid)
	})
}

func TestCreateCourierCommand_Validate_Success(t *testing.T) {
	// Arrange
	name := "Valid Courier"
	speed := 3
	location, err := kernel.NewLocation(5, 7)
	require.NoError(t, err)

	cmd, err := commands.NewCreateCourierCommand(name, speed, location)
	require.NoError(t, err)

	// Act
	err = cmd.Validate()

	// Assert
	assert.NoError(t, err)
}

func TestCreateCourierCommand_Validate_ZeroValue(t *testing.T) {
	// Arrange - zero value command (not constructed via constructor)
	var cmd commands.CreateCourierCommand

	// Act
	err := cmd.Validate()

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, commands.ErrCreateCourierCommandIsNotConstructed)
	assert.Equal(t,
		"CreateCourierCommand must be created via NewCreateCourierCommand constructor",
		commands.ErrCreateCourierCommandIsNotConstructed.Error(),
	)
}

func TestCreateCourierCommand_Validate_ErrorType(t *testing.T) {
	// Test that the specific error type is returned
	t.Run("validation error type verification", func(t *testing.T) {
		var cmd commands.CreateCourierCommand

		err := cmd.Validate()

		require.Error(t, err)
		require.ErrorIs(t, err, commands.ErrCreateCourierCommandIsNotConstructed)
		assert.Contains(t, err.Error(), "CreateCourierCommand must be created via NewCreateCourierCommand constructor")
	})
}

// Benchmark test to ensure performance is acceptable.
func BenchmarkNewCreateCourierCommand(b *testing.B) {
	location, err := kernel.NewLocation(5, 7)
	require.NoError(b, err)
	name := "Benchmark Courier"
	speed := 3

	b.ResetTimer()
	for range b.N {
		_, benchErr := commands.NewCreateCourierCommand(name, speed, location)
		if benchErr != nil {
			b.Fatal(benchErr)
		}
	}
}
