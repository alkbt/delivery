// Package courier provides domain entities and business logic for courier management
// in the delivery system. It implements the Courier aggregate root with movement
// capabilities, order management, and storage place handling.
//
// The package includes:
//   - Courier: The aggregate root that manages courier identity, movement, and orders
//   - StoragePlace: An entity that manages temporary storage of orders during delivery
//
// Key business rules:
//   - Couriers must have a valid unique identifier, name, and speed
//   - Couriers can pick up and deliver orders based on location and capacity
//   - Storage places enforce volume constraints and can store at most one order
//   - Couriers can only take orders that fit in their available storage places
//
// The package follows Domain-Driven Design principles, providing rich domain
// behavior, encapsulation, and validation to ensure business rules are enforced.
package courier
