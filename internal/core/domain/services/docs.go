// Package services provides domain services that orchestrate business operations
// across multiple domain entities in the delivery system. It implements complex
// business workflows that don't naturally belong to a single aggregate root.
//
// The package includes:
//   - OrderDispatcher: A domain service for finding and assigning couriers to orders
//
// Domain services coordinate between aggregates, implementing business logic that
// spans multiple bounded contexts following Domain-Driven Design principles.
package services
