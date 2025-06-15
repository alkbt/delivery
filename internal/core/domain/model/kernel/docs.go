// Package kernel provides core domain primitives and utilities for the delivery system.
// It implements fundamental building blocks following Domain-Driven Design principles
// that are used throughout the domain model.
//
// The package includes:
//   - UUID: A value object for unique identifiers with validation and comparison capabilities
//   - Location: A value object representing coordinates on the delivery grid
//   - ConstructorGuard: A defensive programming pattern to ensure proper object construction
//
// These primitives enforce domain invariants and validation rules, ensuring that
// domain objects are always in a valid state. They are designed to be immutable
// and thread-safe, making them suitable for concurrent use.
//
// The package follows Domain-Driven Design best practices, providing rich domain
// behavior and encapsulation of implementation details.
package kernel
