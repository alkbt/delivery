package ports

import (
	"context"
)

// UnitOfWorkFactory creates new UnitOfWork instances for each request/command.
// This ensures proper isolation between concurrent operations.
type UnitOfWorkFactory interface {
	Create() UnitOfWork
}

// UnitOfWork represents a business transaction boundary.
// It provides transaction control and tracks aggregate changes.
// Client code must explicitly manage transaction lifecycle.
type UnitOfWork interface {
	// Begin starts a new database transaction.
	Begin(ctx context.Context) error

	// Commit commits the current transaction.
	// Returns error if no active transaction or commit fails.
	Commit(ctx context.Context) error

	// Rollback rolls back the current transaction.
	// Returns error if no active transaction or rollback fails.
	Rollback(ctx context.Context) error

	// CourierRepository returns a CourierRepository instance bound to the current transaction.
	// Repository will use the transaction started by Begin().
	CourierRepository() CourierRepository

	// OrderRepository returns an OrderRepository instance bound to the current transaction.
	// Repository will use the transaction started by Begin().
	OrderRepository() OrderRepository
}
