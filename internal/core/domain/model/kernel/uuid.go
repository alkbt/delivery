package kernel

import (
	"fmt"

	"delivery/internal/pkg/errs"

	"github.com/google/uuid"
)

// ErrUUIDIsNotConstructed indicates that a UUID was not properly initialized through one of the constructor functions.
// This error is returned when validating a zero-value UUID.
var ErrUUIDIsNotConstructed = errs.NewValueIsRequiredError("UUID must be created via NewUUID, UUIDFromString, or UUIDFromBytes")

// UUID is a value object that represents a universally unique identifier.
// It wraps the github.com/google/uuid implementation to provide domain-specific behavior
// and ensure immutability. UUID is designed to be used as an identifier for entities
// and aggregates in Domain-Driven Design.
//
// The zero value of UUID is invalid and must be constructed using one of the provided
// factory functions: NewUUID, UUIDFromString, or UUIDFromBytes.
//
// UUID is immutable and thread-safe, making it suitable for concurrent use.
//
// Example usage:
//
//	// Create a new random UUID
//	id := kernel.NewUUID()
//
//	// Create from string representation
//	id, err := kernel.UUIDFromString("550e8400-e29b-41d4-a716-446655440000")
//	if err != nil {
//	    // handle error
//	}
//
//	// Use as entity identifier
//	type Order struct {
//	    ID kernel.UUID
//	    // other fields...
//	}
type UUID struct {
	id uuid.UUID
}

// NewUUID generates a new random UUID (version 4).
// This is the primary way to create new identifiers for entities.
// The generated UUID is guaranteed to be valid and unique with
// extremely high probability.
//
// Example:
//
//	orderID := kernel.NewUUID()
//	fmt.Println(orderID.String()) // e.g., "550e8400-e29b-41d4-a716-446655440000"
func NewUUID() UUID {
	return UUID{
		id: uuid.New(),
	}
}

// UUIDFromString parses a UUID from its string representation.
// It accepts standard UUID formats including:
//   - "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
//   - "{6ba7b810-9dad-11d1-80b4-00c04fd430c8}"
//   - "urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8"
//
// Returns an error if the string is not a valid UUID format.
// This function is typically used when reconstructing entities from
// persistence or when parsing UUIDs from external systems.
//
// Example:
//
//	id, err := kernel.UUIDFromString("550e8400-e29b-41d4-a716-446655440000")
//	if err != nil {
//	    return fmt.Errorf("invalid order ID: %w", err)
//	}
func UUIDFromString(s string) (UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return UUID{}, fmt.Errorf("invalid UUID format: %w", err)
	}
	return UUID{id: id}, nil
}

// UUIDFromBytes creates a UUID from a byte slice.
// The byte slice must be exactly 16 bytes long.
// Returns an error if the byte slice is not valid for UUID construction.
//
// This function is useful when working with binary protocols or
// when UUIDs are stored as binary data in databases.
//
// Example:
//
//	bytes := []byte{0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1,
//	                 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}
//	id, err := kernel.UUIDFromBytes(bytes)
//	if err != nil {
//	    return fmt.Errorf("invalid UUID bytes: %w", err)
//	}
func UUIDFromBytes(b []byte) (UUID, error) {
	id, err := uuid.FromBytes(b)
	if err != nil {
		return UUID{}, fmt.Errorf("invalid UUID format: %w", err)
	}
	newID := UUID{id: id}
	if err = newID.Validate(); err != nil {
		return UUID{}, err
	}

	return newID, nil
}

// String returns the standard string representation of the UUID.
// The format is "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" where x is a hexadecimal digit.
// For a zero value UUID, this returns "00000000-0000-0000-0000-000000000000".
//
// This method is commonly used for:
//   - Logging and debugging
//   - Serialization to JSON or other text formats
//   - Display in user interfaces
//   - Storage as text in databases
//
// Example:
//
//	id := kernel.NewUUID()
//	fmt.Printf("Order created with ID: %s\n", id.String())
func (u UUID) String() string {
	return u.id.String()
}

// Bytes returns the underlying UUID value.
// Note: This returns the internal uuid.UUID type, not a byte slice.
// For a byte slice representation, use id.Bytes()[:].
//
// This method provides access to the underlying UUID for cases where
// integration with external libraries or low-level operations are needed.
// However, direct access should be minimized to maintain encapsulation.
//
// Example:
//
//	id := kernel.NewUUID()
//	googleUUID := id.Bytes()
//	byteSlice := googleUUID[:]
func (u UUID) Bytes() uuid.UUID {
	return u.id
}

// IsEqual compares two UUIDs for equality.
// Returns true if both UUIDs represent the same value, false otherwise.
// This comparison is case-insensitive for the hexadecimal digits.
//
// Example:
//
//	id1 := kernel.NewUUID()
//	id2 := kernel.NewUUID()
//	id3 := id1
//
//	fmt.Println(id1.IsEqual(id2)) // false (different UUIDs)
//	fmt.Println(id1.IsEqual(id3)) // true (same UUID)
func (u UUID) IsEqual(other UUID) bool {
	return u.id == other.id
}

// Validate checks if the UUID is properly constructed.
// Returns ErrUUIDIsNotConstructed if the UUID is a zero value (nil UUID).
// A valid UUID is any UUID that was created through one of the constructor functions.
//
// This method is useful for validating domain objects during construction
// or when receiving data from external sources.
//
// Example:
//
//	type Order struct {
//	    ID kernel.UUID
//	}
//
//	func NewOrder(id kernel.UUID) (*Order, error) {
//	    if err := id.Validate(); err != nil {
//	        return nil, fmt.Errorf("invalid order ID: %w", err)
//	    }
//	    return &Order{ID: id}, nil
//	}
func (u UUID) Validate() error {
	if u.id == uuid.Nil {
		return ErrUUIDIsNotConstructed
	}
	return nil
}
