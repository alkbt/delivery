// Package order provides domain entities and business logic for order management
// in the delivery system. It implements the Order aggregate root with lifecycle
// management and state transitions.
//
// The package includes:
//   - Order: The aggregate root that manages order identity, properties, and lifecycle
//   - Status: A state machine that enforces valid order status transitions
//
// Key business rules:
//   - Orders must have a valid unique identifier, location, and positive volume
//   - Order status follows a defined workflow: Created -> Assigned -> Completed
//   - Orders can be reassigned while in the Assigned status
//   - Orders can only be completed when in the Assigned status
//
// The package follows Domain-Driven Design principles, providing rich domain
// behavior, encapsulation, and validation to ensure business rules are enforced.
package order
