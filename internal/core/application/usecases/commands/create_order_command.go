package commands

import (
	"errors"

	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/pkg/guard"
)

var (
	ErrCreateOrderCommandIsNotConstructed = errors.New(
		"CreateOrderCommand must be created via NewCreateOrderCommand constructor",
	)
	ErrStreetIsRequired = errors.New("street is required")
	ErrVolumeIsInvalid  = errors.New("volume must be greater than 0")
)

// CreateOrderCommand represents a request to create a new delivery order.
// Encapsulates order details including destination and package volume requirements.
//
// Example:
//
//	orderID := kernel.NewUUID()
//	cmd, err := NewCreateOrderCommand(orderID, "123 Main Street", 25)
//	if err != nil {
//	    return fmt.Errorf("invalid order data: %w", err)
//	}
//
//	handler := NewCreateOrderCommandHandler(uowFactory)
//	if err := handler.Handle(ctx, cmd); err != nil {
//	    return fmt.Errorf("failed to create order: %w", err)
//	}
//	fmt.Printf("Order %s created and awaiting courier assignment", orderID)
type CreateOrderCommand struct { //nolint:recvcheck //using for validation
	orderID kernel.UUID
	street  string
	volume  int

	guard guard.ConstructorGuard
}

// NewCreateOrderCommand creates a command to register a new delivery order.
// Validates that order ID is valid, street is not empty, and volume is positive.
// Returns an error if any validation fails.
func NewCreateOrderCommand(orderID kernel.UUID, street string, volume int) (CreateOrderCommand, error) {
	orderCommand := CreateOrderCommand{
		guard: guard.NewConstructorGuard(),
	}

	if err := errors.Join(
		orderCommand.setOrderID(orderID),
		orderCommand.setStreet(street),
		orderCommand.setVolume(volume),
	); err != nil {
		return CreateOrderCommand{}, err
	}

	return orderCommand, nil
}

// Validate ensures the command was created through the constructor.
// Returns ErrCreateOrderCommandIsNotConstructed if validation fails.
func (c CreateOrderCommand) Validate() error {
	return c.guard.Validate(ErrCreateOrderCommandIsNotConstructed)
}

// OrderID returns the unique identifier for the order.
func (c CreateOrderCommand) OrderID() kernel.UUID {
	return c.orderID
}

// Street returns the delivery destination street address.
func (c CreateOrderCommand) Street() string {
	return c.street
}

// Volume returns the package volume in cubic units.
func (c CreateOrderCommand) Volume() int {
	return c.volume
}

func (c *CreateOrderCommand) setOrderID(orderID kernel.UUID) error {
	if err := orderID.Validate(); err != nil {
		return err
	}

	c.orderID = orderID
	return nil
}

func (c *CreateOrderCommand) setStreet(street string) error {
	if street == "" {
		return ErrStreetIsRequired
	}

	c.street = street
	return nil
}

func (c *CreateOrderCommand) setVolume(volume int) error {
	if volume <= 0 {
		return ErrVolumeIsInvalid
	}

	c.volume = volume
	return nil
}
