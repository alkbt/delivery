package commands_test

import (
	"testing"

	"delivery/internal/core/application/usecases/commands"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoveCouriersCommand_Validate_WhenConstructedProperly_ShouldReturnNoError(t *testing.T) {
	// Arrange
	cmd := commands.NewMoveCouriersCommand()

	// Act
	err := cmd.Validate()

	// Assert
	require.NoError(t, err)
}

func TestMoveCouriersCommand_Validate_WhenNotConstructed_ShouldReturnError(t *testing.T) {
	// Arrange
	var cmd commands.MoveCouriersCommand // zero-value command

	// Act
	err := cmd.Validate()

	// Assert
	require.Error(t, err)
	assert.Equal(t, commands.ErrMoveCouriersCommandIsNotConstructed, err)
}
