package commands

import (
	"context"
)

// AddCourierStorageCommandHandler handles the business logic for adding storage places to couriers.
// Uses transactional operations to ensure data consistency when modifying courier entities.
//
// Example:
//
//	handler := NewAddCourierStorageCommandHandler(uowFactory)
//	cmd, _ := NewAddCourierStorageCommand(courierID, "Cold Storage", 30)
//	err := handler.Handle(ctx, cmd)
//	if err != nil {
//	    log.Printf("Failed to add storage: %v", err)
//	}
type AddCourierStorageCommandHandler struct {
	uowFactory CourierUoWFactory
}

// NewAddCourierStorageCommandHandler creates a new handler for adding storage to couriers.
// Requires a CourierUoWFactory for transactional operations.
func NewAddCourierStorageCommandHandler(uowFactory CourierUoWFactory) AddCourierStorageCommandHandler {
	return AddCourierStorageCommandHandler{
		uowFactory: uowFactory,
	}
}

// Handle processes the AddCourierStorageCommand within a transaction.
// Retrieves the courier, adds the new storage place, and persists the changes.
// Automatically rolls back on any error to maintain data consistency.
func (h *AddCourierStorageCommandHandler) Handle(ctx context.Context, cmd AddCourierStorageCommand) error {
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
	courierEntity, err := courierRepo.Get(ctx, cmd.CourierID())
	if err != nil {
		return err
	}

	if err = courierEntity.AddStoragePlace(cmd.Name(), cmd.TotalVolume()); err != nil {
		return err
	}

	if err = courierRepo.Update(ctx, courierEntity); err != nil {
		return err
	}

	if err = uow.Commit(ctx); err != nil {
		return err
	}

	return nil
}
