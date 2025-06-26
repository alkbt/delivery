package commands_test

import (
	"testing"

	"delivery/internal/core/application/usecases/commands"
	"delivery/internal/core/domain/model/kernel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCreateOrderCommand_ValidInput(t *testing.T) {
	id := kernel.NewUUID()
	cmd, err := commands.NewCreateOrderCommand(id, "Main St", 10)
	require.NoError(t, err)
	assert.Equal(t, id, cmd.OrderID())
	assert.Equal(t, "Main St", cmd.Street())
	assert.Equal(t, 10, cmd.Volume())
}

func TestNewCreateOrderCommand_InvalidInput(t *testing.T) {
	id := kernel.NewUUID()
	_, err := commands.NewCreateOrderCommand(id, "", 0)
	require.Error(t, err)
}

func TestNewCreateOrderCommand_InvalidOrderID(t *testing.T) {
	invalidID := kernel.UUID{} // zero value, should trigger validation error
	_, err := commands.NewCreateOrderCommand(invalidID, "Main St", 10)
	require.Error(t, err)
	assert.ErrorIs(t, err, kernel.ErrUUIDIsNotConstructed)
}

func TestNewCreateOrderCommand_EmptyStreet(t *testing.T) {
	id := kernel.NewUUID()
	_, err := commands.NewCreateOrderCommand(id, "", 10)
	require.Error(t, err)
	assert.ErrorIs(t, err, commands.ErrStreetIsRequired)
}

func TestNewCreateOrderCommand_InvalidVolume(t *testing.T) {
	id := kernel.NewUUID()
	_, err := commands.NewCreateOrderCommand(id, "Main St", 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, commands.ErrVolumeIsInvalid)
}
