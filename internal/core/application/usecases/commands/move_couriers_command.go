package commands

import (
	"errors"

	"delivery/internal/pkg/guard"
)

// MoveCouriersCommand triggers movement of all assigned couriers towards their destinations.
// This batch operation updates courier positions and completes orders when destinations are reached.
//
// Example:
//
//	cmd := NewMoveCouriersCommand()
//	handler := NewMoveCouriersCommandHandler(uowFactory)
//
//	// Run periodically to simulate courier movement
//	ticker := time.NewTicker(5 * time.Second)
//	for range ticker.C {
//	    if err := handler.Handle(ctx, cmd); err != nil {
//	        log.Printf("Movement update failed: %v", err)
//	    }
//	}
type MoveCouriersCommand struct {
	guard guard.ConstructorGuard
}

var (
	ErrMoveCouriersCommandIsNotConstructed = errors.New(
		"MoveCouriersCommand must be created via NewMoveCouriersCommand constructor",
	)
)

// NewMoveCouriersCommand creates a command to trigger courier movement updates.
// This is a parameterless command that processes all active deliveries.
func NewMoveCouriersCommand() MoveCouriersCommand {
	command := MoveCouriersCommand{
		guard: guard.NewConstructorGuard(),
	}

	return command
}

// Validate ensures the command was created through the constructor.
// Returns ErrMoveCouriersCommandIsNotConstructed if validation fails.
func (c *MoveCouriersCommand) Validate() error {
	return c.guard.Validate(ErrMoveCouriersCommandIsNotConstructed)
}
