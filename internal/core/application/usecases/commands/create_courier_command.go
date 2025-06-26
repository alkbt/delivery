package commands

import (
	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/pkg/guard"
	"errors"
)

var (
	ErrCreateCourierCommandIsNotConstructed = errors.New(
		"CreateCourierCommand must be created via NewCreateCourierCommand constructor",
	)
	ErrNameIsRequired = errors.New("name is required")
	ErrSpeedIsInvalid = errors.New("speed must be greater than 0")
)

// CreateCourierCommand represents a request to register a new courier in the delivery system.
// Encapsulates all data needed to create a courier entity with delivery capabilities.
//
// Example:
//
//	location := kernel.NewLocation(55.7558, 37.6173) // Moscow coordinates
//	cmd, err := NewCreateCourierCommand("John Doe", 60, location)
//	if err != nil {
//	    return fmt.Errorf("invalid courier data: %w", err)
//	}
//
//	handler := NewCreateCourierCommandHandler(uowFactory)
//	if err := handler.Handle(ctx, cmd); err != nil {
//	    return fmt.Errorf("failed to create courier: %w", err)
//	}
//	fmt.Printf("Created courier with ID: %s", cmd.CourierID())
type CreateCourierCommand struct { //nolint:recvcheck //using for validation
	courierID kernel.UUID
	name      string
	speed     int
	location  kernel.Location

	guard guard.ConstructorGuard
}

// NewCreateCourierCommand creates a command to register a new courier.
// Automatically generates a unique ID for the courier.
// Validates that name is not empty, speed is positive, and location is valid.
func NewCreateCourierCommand(name string, speed int, location kernel.Location) (CreateCourierCommand, error) {
	command := CreateCourierCommand{
		guard: guard.NewConstructorGuard(),
	}

	if err := errors.Join(
		command.setCourierID(kernel.NewUUID()),
		command.setName(name),
		command.setSpeed(speed),
		command.setLocation(location),
	); err != nil {
		return CreateCourierCommand{}, err
	}

	return command, nil
}

// Validate ensures the command was created through the constructor.
// Returns ErrCreateCourierCommandIsNotConstructed if validation fails.
func (c CreateCourierCommand) Validate() error {
	return c.guard.Validate(ErrCreateCourierCommandIsNotConstructed)
}

// CourierID returns the courier ID from the command.
func (c CreateCourierCommand) CourierID() kernel.UUID {
	return c.courierID
}

// Name returns the courier name from the command.
func (c CreateCourierCommand) Name() string {
	return c.name
}

// Speed returns the courier speed from the command.
func (c CreateCourierCommand) Speed() int {
	return c.speed
}

// Location returns the courier location from the command.
func (c CreateCourierCommand) Location() kernel.Location {
	return c.location
}

func (c *CreateCourierCommand) setCourierID(id kernel.UUID) error {
	if err := id.Validate(); err != nil {
		return err
	}

	c.courierID = id
	return nil
}

func (c *CreateCourierCommand) setName(name string) error {
	if name == "" {
		return ErrNameIsRequired
	}

	c.name = name
	return nil
}

func (c *CreateCourierCommand) setSpeed(speed int) error {
	if speed <= 0 {
		return ErrSpeedIsInvalid
	}

	c.speed = speed
	return nil
}

func (c *CreateCourierCommand) setLocation(location kernel.Location) error {
	if err := location.Validate(); err != nil {
		return err
	}

	c.location = location
	return nil
}
