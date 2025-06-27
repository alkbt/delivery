package commands

import (
	"context"
	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/core/domain/model/order"
)

// CreateOrderCommandHandler handles the business logic for order creation.
// Creates new orders with random delivery locations and initial "created" status.
//
// Example:
//
//	handler := NewCreateOrderCommandHandler(uowFactory)
//	orderID := kernel.NewUUID()
//	cmd, _ := NewCreateOrderCommand(orderID, "456 Oak Avenue", 15)
//
//	if err := handler.Handle(ctx, cmd); err != nil {
//	    return fmt.Errorf("order creation failed: %w", err)
//	}
//	// Order is now created and ready for courier assignment
type CreateOrderCommandHandler struct {
	uowFactory OrderUoWFactory
}

// NewCreateOrderCommandHandler creates a handler for order creation operations.
// Requires an OrderUoWFactory for transactional persistence.
func NewCreateOrderCommandHandler(uowFactory OrderUoWFactory) CreateOrderCommandHandler {
	return CreateOrderCommandHandler{
		uowFactory: uowFactory,
	}
}

// Handle processes the order creation command.
// Generates a random delivery location and creates the order in "created" status.
// Uses transaction to ensure order is properly persisted or rolled back on error.
func (h *CreateOrderCommandHandler) Handle(ctx context.Context, cmd CreateOrderCommand) error {
	if err := cmd.Validate(); err != nil {
		return err
	}

	location, err := kernel.NewRandomLocation()
	if err != nil {
		return err
	}

	uow := h.uowFactory.Create()
	if err = uow.Begin(ctx); err != nil {
		return err
	}

	defer func() {
		_ = uow.Rollback(ctx)
	}()

	orderRepo := uow.OrderRepository()
	order, err := order.NewOrder(cmd.OrderID(), location, cmd.Volume())
	if err != nil {
		return err
	}

	if err = orderRepo.Add(ctx, order); err != nil {
		return err
	}

	if err = uow.Commit(ctx); err != nil {
		return err
	}

	return nil
}
