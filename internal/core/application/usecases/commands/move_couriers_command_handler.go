package commands

import (
	"context"
	"delivery/internal/core/domain/model/courier"
	"delivery/internal/core/domain/model/order"
)

// MoveCouriersCommandHandler orchestrates the movement of all active couriers.
// Processes each assigned order, moves couriers towards destinations, and completes
// deliveries when couriers reach their targets.
//
// Example:
//
//	handler := NewMoveCouriersCommandHandler(uowFactory)
//	cmd := NewMoveCouriersCommand()
//
//	// Execute movement update
//	if err := handler.Handle(ctx, cmd); err != nil {
//	    return fmt.Errorf("courier movement failed: %w", err)
//	}
//
//	// This would typically be called periodically by a scheduler
type MoveCouriersCommandHandler struct {
	uowFactory UoWFactory
}

// NewMoveCouriersCommandHandler creates a handler for courier movement operations.
// Requires a UoWFactory for coordinating updates across order and courier repositories.
func NewMoveCouriersCommandHandler(uowFactory UoWFactory) MoveCouriersCommandHandler {
	return MoveCouriersCommandHandler{
		uowFactory: uowFactory,
	}
}

// Handle processes the courier movement command.
// Retrieves all orders in "assigned" status, moves each courier towards its destination,
// and completes orders when couriers arrive. All updates occur within a single transaction.
func (h *MoveCouriersCommandHandler) Handle(ctx context.Context, cmd MoveCouriersCommand) error {
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
	ordersRepo := uow.OrderRepository()

	orders, err := ordersRepo.GetAllInAssignedStatus(ctx)
	if err != nil {
		return err
	}

	for _, order := range orders {
		courier, courierErr := courierRepo.Get(ctx, *order.Courier())
		if courierErr != nil {
			return courierErr
		}

		if err = h.moveOrderCourier(order, courier); err != nil {
			return err
		}

		if err = ordersRepo.Update(ctx, order); err != nil {
			return err
		}

		if err = courierRepo.Update(ctx, courier); err != nil {
			return err
		}
	}

	if err = uow.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// moveOrderCourier handles the movement logic for a single courier-order pair.
// Moves the courier towards the order location and completes both order and courier
// states when the destination is reached.
func (h *MoveCouriersCommandHandler) moveOrderCourier(
	order *order.Order,
	courier *courier.Courier,
) error {
	if err := courier.Move(order.Location()); err != nil {
		return err
	}

	if equal, err := courier.Location().IsEqual(order.Location()); err != nil || !equal {
		return err
	}

	if err := order.Complete(); err != nil {
		return err
	}

	if err := courier.CompleteOrder(order.ID()); err != nil {
		return err
	}

	return nil
}
