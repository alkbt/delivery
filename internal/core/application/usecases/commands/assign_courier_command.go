package commands

import (
	"delivery/internal/pkg/guard"
	"errors"
)

var ErrAssignCourierCommandIsNotConstructed = errors.New(
	"AssignCourierCommand must be created via NewAssignCourierCommand constructor",
)

// AssignCourierCommand triggers the assignment of an available courier to a pending order.
// This command represents the business operation of matching delivery resources with orders.
// It finds the first order in "created" status and assigns the most suitable courier.
//
// Example:
//
//	cmd := NewAssignCourierCommand()
//	handler := NewAssignCourierCommandHandler(uowFactory, dispatcher)
//	err := handler.Handle(ctx, cmd)
//	if err != nil {
//	    log.Printf("No orders to assign or no available couriers: %v", err)
//	}
type AssignCourierCommand struct {
	guard guard.ConstructorGuard
}

// NewAssignCourierCommand creates a new command to trigger courier assignment.
// This is a parameterless command that initiates the courier-order matching process.
func NewAssignCourierCommand() AssignCourierCommand {
	return AssignCourierCommand{
		guard: guard.NewConstructorGuard(),
	}
}

// Validate ensures the command was created through the constructor.
// Returns ErrAssignCourierCommandIsNotConstructed if validation fails.
func (c *AssignCourierCommand) Validate() error {
	return c.guard.Validate(
		ErrAssignCourierCommandIsNotConstructed,
	)
}
