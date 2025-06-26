package commands

import (
	"errors"

	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/pkg/guard"
)

var (
	ErrAddCourierStorageCommandIsNotConstructed = errors.New(
		"AddCourierStorageCommand must be created via NewAddCourierStorageCommand constructor",
	)
	ErrTotalVolumeIsInvalid = errors.New("total volume must be greater than 0")
)

// AddCourierStorageCommand represents a request to add a new storage place to an existing courier.
// This command encapsulates the business operation of expanding a courier's delivery capacity
// by adding additional storage compartments.
//
// Example:
//
//	courierID := kernel.MustNewUUID("550e8400-e29b-41d4-a716-446655440000")
//	cmd, err := NewAddCourierStorageCommand(courierID, "Insulated Box", 50)
//	if err != nil {
//	    return fmt.Errorf("invalid command: %w", err)
//	}
//
//	handler := NewAddCourierStorageCommandHandler(uowFactory)
//	if err := handler.Handle(ctx, cmd); err != nil {
//	    return fmt.Errorf("failed to add storage: %w", err)
//	}
type AddCourierStorageCommand struct { //nolint:recvcheck //using for validation
	courierID   kernel.UUID
	name        string
	totalVolume int

	guard guard.ConstructorGuard
}

// NewAddCourierStorageCommand creates a new command to add storage to a courier.
// Validates that the courier ID is valid, name is not empty, and volume is positive.
// Returns an error if any validation fails.
func NewAddCourierStorageCommand(
	courierID kernel.UUID,
	name string,
	totalVolume int,
) (AddCourierStorageCommand, error) {
	command := AddCourierStorageCommand{
		guard: guard.NewConstructorGuard(),
	}

	if err := errors.Join(
		command.setCourierID(courierID),
		command.setName(name),
		command.setTotalVolume(totalVolume),
	); err != nil {
		return AddCourierStorageCommand{}, err
	}

	return command, nil
}

// Validate ensures the command was created through the constructor.
// Returns ErrAddCourierStorageCommandIsNotConstructed if validation fails.
func (c AddCourierStorageCommand) Validate() error {
	return c.guard.Validate(ErrAddCourierStorageCommandIsNotConstructed)
}

// CourierID returns the ID of the courier to add storage to.
func (c AddCourierStorageCommand) CourierID() kernel.UUID {
	return c.courierID
}

// Name returns the name of the storage place to be added.
func (c AddCourierStorageCommand) Name() string {
	return c.name
}

// TotalVolume returns the total volume capacity of the storage place.
func (c AddCourierStorageCommand) TotalVolume() int {
	return c.totalVolume
}

func (c *AddCourierStorageCommand) setCourierID(courierID kernel.UUID) error {
	if err := courierID.Validate(); err != nil {
		return err
	}

	c.courierID = courierID
	return nil
}

func (c *AddCourierStorageCommand) setName(name string) error {
	if name == "" {
		return ErrNameIsRequired
	}

	c.name = name
	return nil
}

func (c *AddCourierStorageCommand) setTotalVolume(totalVolume int) error {
	if totalVolume <= 0 {
		return ErrTotalVolumeIsInvalid
	}

	c.totalVolume = totalVolume
	return nil
}
