package commands

import (
	"context"

	"delivery/internal/core/domain/model/courier"
)

// CreateCourierCommandHandler handles the business logic for courier registration.
// Creates and persists new courier entities with initial location and capabilities.
//
// Example:
//
//	handler := NewCreateCourierCommandHandler(uowFactory)
//	location := kernel.NewLocation(40.7128, -74.0060) // New York
//	cmd, _ := NewCreateCourierCommand("Express Courier", 80, location)
//
//	if err := handler.Handle(ctx, cmd); err != nil {
//	    return fmt.Errorf("courier registration failed: %w", err)
//	}
type CreateCourierCommandHandler struct {
	uowFactory CourierUoWFactory
}

// NewCreateCourierCommandHandler creates a handler for courier registration.
// Requires a CourierUoWFactory for transactional persistence operations.
func NewCreateCourierCommandHandler(uowFactory CourierUoWFactory) CreateCourierCommandHandler {
	return CreateCourierCommandHandler{
		uowFactory: uowFactory,
	}
}

// Handle processes the courier creation command.
// Creates a new courier entity and persists it within a transaction.
// Automatically rolls back on any error to prevent partial data.
func (h *CreateCourierCommandHandler) Handle(ctx context.Context, cmd CreateCourierCommand) error {
	if err := cmd.Validate(); err != nil {
		return err
	}

	uow := h.uowFactory.Create()
	if err := uow.Begin(ctx); err != nil {
		return err
	}

	defer func() {
		_ = uow.Rollback(ctx)
	}()

	courierRepo := uow.CourierRepository()
	courierEntity, err := courier.NewCourier(cmd.CourierID(), cmd.Name(), cmd.Speed(), cmd.Location())
	if err != nil {
		return err
	}

	if err = courierRepo.Add(ctx, courierEntity); err != nil {
		return err
	}

	if err = uow.Commit(ctx); err != nil {
		return err
	}

	return nil
}
