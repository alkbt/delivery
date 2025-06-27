// Package commands contains business operations that modify system state.
// Implements the Command pattern for write operations in the CQRS architecture.
// All commands follow a consistent pattern: validation, transaction management, and persistence.
package commands

import (
	"context"

	"delivery/internal/core/ports"
)

// Unit of Work interfaces provide transaction management for command handlers.
// These abstractions ensure data consistency across aggregate boundaries.
type (
	// TxManager handles database transaction lifecycle.
	// Ensures atomic operations across multiple repository calls.
	TxManager interface {
		Begin(ctx context.Context) error
		Commit(ctx context.Context) error
		Rollback(ctx context.Context) error
	}

	// OrderRepoFactory provides access to order repository within a transaction.
	OrderRepoFactory interface {
		OrderRepository() ports.OrderRepository
	}

	// CourierRepoFactory provides access to courier repository within a transaction.
	CourierRepoFactory interface {
		CourierRepository() ports.CourierRepository
	}

	// OrderUoW manages transactions for order-only operations.
	// Used when commands only modify order aggregates.
	OrderUoW interface {
		TxManager
		OrderRepoFactory
	}

	// OrderUoWFactory creates new order unit of work instances.
	OrderUoWFactory interface {
		Create() OrderUoW
	}

	// CourierUoW manages transactions for courier-only operations.
	// Used when commands only modify courier aggregates.
	CourierUoW interface {
		TxManager
		CourierRepoFactory
	}

	// CourierUoWFactory creates new courier unit of work instances.
	CourierUoWFactory interface {
		Create() CourierUoW
	}

	// UoW manages transactions across both order and courier aggregates.
	// Used for commands that coordinate changes between multiple aggregate types.
	//
	// Example:
	//   uow := factory.Create()
	//   err := uow.Begin(ctx)
	//   defer uow.Rollback(ctx)
	//
	//   orderRepo := uow.OrderRepository()
	//   courierRepo := uow.CourierRepository()
	//   // ... perform operations
	//
	//   err = uow.Commit(ctx)
	UoW interface {
		TxManager
		CourierRepoFactory
		OrderRepoFactory
	}

	// UoWFactory creates new unit of work instances for cross-aggregate operations.
	UoWFactory interface {
		Create() UoW
	}
)
