package commands

import (
	"context"
	"delivery/internal/core/domain/services"
	"delivery/internal/pkg/errs"
	"errors"
)

var (
	ErrNoFreeCouriersFound = errors.New("no free couriers found")
	ErrNoOrderFound        = errors.New("no order found")
)

// AssignCourierCommandHandler orchestrates the courier assignment process.
// Finds pending orders and matches them with available couriers using business rules.
// Ensures transactional consistency when updating both order and courier states.
//
// Example:
//
//	handler := NewAssignCourierCommandHandler(uowFactory)
//	cmd := NewAssignCourierCommand()
//	err := handler.Handle(ctx, cmd)
//	switch {
//	case errors.Is(err, ErrNoOrderFound):
//	    log.Println("No pending orders")
//	case errors.Is(err, ErrNoFreeCouriersFound):
//	    log.Println("All couriers are busy")
//	case err != nil:
//	    log.Printf("Assignment failed: %v", err)
//	default:
//	    log.Println("Courier assigned successfully")
//	}
type AssignCourierCommandHandler struct {
	uowFactory UoWFactory
}

// NewAssignCourierCommandHandler creates a handler for courier assignment operations.
// Requires a UoWFactory for coordinating transactional updates across repositories.
func NewAssignCourierCommandHandler(uowFactory UoWFactory) AssignCourierCommandHandler {
	return AssignCourierCommandHandler{
		uowFactory: uowFactory,
	}
}

// Handle processes the courier assignment command.
// Retrieves the first pending order, finds available couriers, and uses OrderDispatcher
// to select the best match. Updates both entities within a single transaction.
// Returns specific errors for no orders (ErrNoOrderFound) or no couriers (ErrNoFreeCouriersFound).
func (h AssignCourierCommandHandler) Handle(ctx context.Context, command AssignCourierCommand) error {
	if err := command.Validate(); err != nil {
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

	order, err := ordersRepo.GetFirstInCreatedStatus(ctx)
	if errors.Is(err, errs.ErrObjectNotFound) {
		return ErrNoOrderFound
	}
	if err != nil {
		return err
	}

	couriers, err := courierRepo.GetAllFree(ctx)
	if err != nil {
		return err
	}
	if len(couriers) == 0 {
		return ErrNoFreeCouriersFound
	}

	assignedCourier, err := services.NewOrderDispatcher().Dispatch(order, couriers)
	if err != nil {
		return err
	}

	if err = ordersRepo.Update(ctx, order); err != nil {
		return err
	}

	if err = courierRepo.Update(ctx, assignedCourier); err != nil {
		return err
	}

	if err = uow.Commit(ctx); err != nil {
		return err
	}

	return nil
}
