package commands_test

import (
	"testing"

	"delivery/internal/core/application/usecases/commands"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAssignCourierCommand_Success(t *testing.T) {
	// Act
	cmd := commands.NewAssignCourierCommand()

	// Assert
	assert.NotZero(t, cmd)
	require.NoError(t, cmd.Validate())
}

func TestAssignCourierCommand_Validate_ZeroValue(t *testing.T) {
	// Arrange
	var cmd commands.AssignCourierCommand // zero value, not constructed via constructor

	// Act
	err := cmd.Validate()

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, commands.ErrAssignCourierCommandIsNotConstructed)
}
