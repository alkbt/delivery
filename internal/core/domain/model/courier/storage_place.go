package courier

import (
	"errors"
	"fmt"

	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/pkg/errs"
)

var (
	// ErrCannotStoreOrderInThisStoragePlace indicates that an order cannot be stored
	// in the storage place due to insufficient capacity or the place being occupied.
	ErrCannotStoreOrderInThisStoragePlace = errors.New("cannot store order in this storage place")

	// ErrOrderNotStoredInThisPlace indicates that the specified order is not
	// currently stored in this storage place, either because the place is empty
	// or contains a different order.
	ErrOrderNotStoredInThisPlace = errors.New("order not stored in this place")

	// ErrStoragePlaceIsNotConstructed indicates that the StoragePlace was not
	// properly initialized through the NewStoragePlace constructor function.
	ErrStoragePlaceIsNotConstructed = errors.New("StoragePlace must be created via NewStoragePlace constructor")
)

// StoragePlace represents a physical storage location where orders can be temporarily stored
// during the delivery process. It is a domain entity that encapsulates the business rules
// and constraints for order storage operations.
//
// A StoragePlace has a fixed total volume capacity and can store at most one order at a time.
// The entity enforces business rules such as volume validation, occupancy management,
// and proper construction validation.
//
// Key business rules:
//   - Must be constructed through NewStoragePlace constructor
//   - Can only store one order at a time (binary occupancy)
//   - Order volume must not exceed storage place capacity
//   - Only the stored order can be cleared from the storage place
//
// Example usage:
//
//	place, err := courier.NewStoragePlace(
//	    kernel.NewUUID(),
//	    "Warehouse A - Bay 1",
//	    1000, // volume capacity
//	)
//	if err != nil {
//	    return err
//	}
//
//	// Check if order can be stored
//	canStore, err := place.CanStore(500)
//	if err != nil {
//	    return err
//	}
//
//	// Store the order
//	if canStore {
//	    err = place.Store(orderID, 500)
//	}
type StoragePlace struct {
	// id uniquely identifies the storage place
	id kernel.UUID

	// name is a human-readable identifier for the storage place
	name string

	// totalVolume represents the maximum volume capacity of the storage place
	totalVolume int

	// orderID points to the currently stored order, nil if empty
	orderID *kernel.UUID

	// guard ensures the entity was properly initialized
	guard kernel.ConstructorGuard
}

// NewStoragePlace creates a new StoragePlace entity with the specified parameters.
// This is the only way to create a properly initialized StoragePlace instance.
//
// The constructor validates all input parameters and ensures the entity is in a
// consistent state before returning. All validation errors are aggregated and
// returned as a single error.
//
// Parameters:
//   - id: Unique identifier for the storage place (must be valid UUID)
//   - name: Human-readable name for the storage place (must not be empty)
//   - totalVolume: Maximum volume capacity (must be greater than 0)
//
// Returns:
//   - *StoragePlace: Properly initialized storage place entity
//   - error: Aggregated validation errors, if any
//
// Example:
//
//	place, err := courier.NewStoragePlace(
//	    kernel.NewUUID(),
//	    "Main Warehouse - Section A",
//	    2000,
//	)
//	if err != nil {
//	    return fmt.Errorf("failed to create storage place: %w", err)
//	}
func NewStoragePlace(id kernel.UUID, name string, totalVolume int) (*StoragePlace, error) {
	place := &StoragePlace{
		guard: kernel.NewConstructorGuard(),
	}

	if err := errors.Join(place.setID(id), place.setName(name), place.setTotalVolume(totalVolume)); err != nil {
		return nil, err
	}

	return place, nil
}

// RestoreStoragePlace reconstructs a StoragePlace entity from persistent storage.
// Unlike NewStoragePlace which creates empty storage places, this constructor restores
// a storage place to its previously persisted state, including any stored order.
//
// This function enables loading storage places from the database while preserving
// their operational state at the time of persistence. The restored storage place
// behaves identically to one created through normal domain operations.
//
// Parameters:
//   - id: Unique identifier for the storage place
//   - name: Human-readable name for the storage place
//   - totalVolume: Maximum volume capacity
//   - orderID: ID of currently stored order (nil if empty)
//
// Returns:
//   - *StoragePlace: Restored storage place entity
//   - error: Validation error if any parameter is invalid
//
// Business Rules:
//   - Storage place ID must be valid
//   - Name cannot be empty
//   - Total volume must be positive
//   - Order ID, if provided, must be valid
//
// Examples:
//
//	// Restore empty storage place
//	place, err := RestoreStoragePlace(id, "Main Bag", 1000, nil)
//	if err != nil {
//	    return fmt.Errorf("restoration failed: %w", err)
//	}
//
//	// Restore occupied storage place
//	place, err := RestoreStoragePlace(id, "Side Pouch", 500, &orderID)
//	if err != nil {
//	    return fmt.Errorf("restoration failed: %w", err)
//	}
func RestoreStoragePlace(id kernel.UUID, name string, totalVolume int, orderID *kernel.UUID) (*StoragePlace, error) {
	place := &StoragePlace{
		guard: kernel.NewConstructorGuard(),
	}

	if err := errors.Join(
		place.setID(id),
		place.setName(name),
		place.setTotalVolume(totalVolume),
		place.setOrderID(orderID),
	); err != nil {
		return nil, err
	}

	return place, nil
}

// IsEqual compares two StoragePlace entities for equality based on their unique identifiers.
// Two storage places are considered equal if they have the same ID, following DDD principles
// where entity equality is determined by identity, not by attribute values.
//
// Parameters:
//   - other: The StoragePlace to compare with (can be nil)
//
// Returns:
//   - bool: True if both storage places have the same ID, false otherwise
//
// Example:
//
//	place1, _ := courier.NewStoragePlace(id, "Place 1", 1000)
//	place2, _ := courier.NewStoragePlace(id, "Place 2", 2000) // Same ID, different attributes
//	fmt.Println(place1.IsEqual(place2)) // true - same identity
func (s *StoragePlace) IsEqual(other *StoragePlace) bool {
	return other != nil && s.id.IsEqual(other.id)
}

// ID returns the unique identifier of the storage place.
// This identifier is immutable and set during construction.
//
// Returns:
//   - kernel.UUID: The unique identifier of this storage place
func (s *StoragePlace) ID() kernel.UUID {
	return s.id
}

// Name returns the human-readable name of the storage place.
// This name is typically used for display purposes and logging.
//
// Returns:
//   - string: The name of this storage place
func (s *StoragePlace) Name() string {
	return s.name
}

// TotalVolume returns the maximum volume capacity of the storage place.
// This represents the total space available for storing orders.
//
// Returns:
//   - int: The maximum volume capacity in cubic units
func (s *StoragePlace) TotalVolume() int {
	return s.totalVolume
}

// OrderID returns the ID of the currently stored order, if any.
// Returns nil if the storage place is currently empty.
//
// Returns:
//   - *kernel.UUID: Pointer to the stored order's ID, or nil if empty
func (s *StoragePlace) OrderID() *kernel.UUID {
	return s.orderID
}

// CanStore determines whether an order with the specified volume can be stored
// in this storage place. This method checks both the availability of the storage
// place and whether it has sufficient capacity.
//
// Business rules enforced:
//   - Volume must be positive (greater than 0)
//   - Storage place must not be currently occupied
//   - Available volume must be sufficient for the order
//
// Parameters:
//   - volume: The volume of the order to be stored (must be > 0)
//
// Returns:
//   - bool: True if the order can be stored, false otherwise
//   - error: Validation error if volume is invalid
//
// Example:
//
//	canStore, err := place.CanStore(500)
//	if err != nil {
//	    return fmt.Errorf("validation failed: %w", err)
//	}
//	if !canStore {
//	    return errors.New("insufficient capacity or place occupied")
//	}
func (s *StoragePlace) CanStore(volume int) (bool, error) {
	if volume <= 0 {
		return false, errs.NewValueIsInvalidErrorWithCause(
			"volume is invalid",
			fmt.Errorf("%d is not greater than 0", volume),
		)
	}

	return !s.isOccupied() && s.totalVolume >= volume, nil
}

// Store places an order in this storage place, marking it as occupied.
// This operation validates the order ID and checks storage constraints before
// proceeding with the storage operation.
//
// Business rules enforced:
//   - Order ID must be valid (properly constructed UUID)
//   - Storage place must have sufficient capacity
//   - Storage place must not be currently occupied
//   - Volume must be positive
//
// Parameters:
//   - orderID: Valid UUID identifying the order to store
//   - volume: Volume of the order (must be > 0 and <= totalVolume)
//
// Returns:
//   - error: Validation or business rule violation error
//
// Example:
//
//	orderID := kernel.NewUUID()
//	err := place.Store(orderID, 750)
//	if err != nil {
//	    if errors.Is(err, courier.ErrCannotStoreOrderInThisStoragePlace) {
//	        return errors.New("storage place is full or occupied")
//	    }
//	    return fmt.Errorf("failed to store order: %w", err)
//	}
func (s *StoragePlace) Store(orderID kernel.UUID, volume int) error {
	if err := orderID.Validate(); err != nil {
		return err
	}

	canStore, err := s.CanStore(volume)
	if err != nil {
		return err
	}

	if !canStore {
		return ErrCannotStoreOrderInThisStoragePlace
	}

	s.orderID = &orderID
	return nil
}

// Clear removes the specified order from this storage place, making it available
// for new orders. This operation validates that the correct order is being removed.
//
// Business rules enforced:
//   - Order ID must be valid (properly constructed UUID)
//   - Storage place must be currently occupied
//   - The stored order ID must match the specified order ID
//
// Parameters:
//   - orderID: Valid UUID of the order to remove (must match stored order)
//
// Returns:
//   - error: Validation error or business rule violation
//
// Example:
//
//	err := place.Clear(storedOrderID)
//	if err != nil {
//	    if errors.Is(err, courier.ErrOrderNotStoredInThisPlace) {
//	        return errors.New("order not found in this storage place")
//	    }
//	    return fmt.Errorf("failed to clear storage place: %w", err)
//	}
func (s *StoragePlace) Clear(orderID kernel.UUID) error {
	if err := orderID.Validate(); err != nil {
		return err
	}

	if !s.isOccupied() || !s.orderID.IsEqual(orderID) {
		return ErrOrderNotStoredInThisPlace
	}

	s.orderID = nil
	return nil
}

func (s *StoragePlace) isOccupied() bool {
	return s.orderID != nil
}

func (s *StoragePlace) setID(id kernel.UUID) error {
	if err := id.Validate(); err != nil {
		return err
	}

	s.id = id
	return nil
}

func (s *StoragePlace) setName(name string) error {
	if name == "" {
		return errs.NewValueIsRequiredError("name is required")
	}

	s.name = name
	return nil
}

func (s *StoragePlace) setTotalVolume(totalVolume int) error {
	if totalVolume <= 0 {
		return errs.NewValueIsInvalidErrorWithCause(
			"totalVolume is invalid",
			fmt.Errorf("%d is not greater than 0", totalVolume),
		)
	}

	s.totalVolume = totalVolume
	return nil
}

// setOrderID sets the stored order ID for this storage place.
// Used during entity restoration to establish the occupied state.
func (s *StoragePlace) setOrderID(orderID *kernel.UUID) error {
	if orderID != nil {
		if err := orderID.Validate(); err != nil {
			return err
		}
	}

	s.orderID = orderID
	return nil
}

// Validate checks if the StoragePlace entity is in a valid state.
// This method ensures the entity was properly constructed through the
// NewStoragePlace constructor function.
//
// Returns:
//   - error: ErrStoragePlaceIsNotConstructed if not properly initialized
//
// Example:
//
//	if err := place.Validate(); err != nil {
//	    return fmt.Errorf("invalid storage place: %w", err)
//	}
func (s *StoragePlace) Validate() error {
	if s == nil {
		return ErrStoragePlaceIsNotConstructed
	}
	return s.guard.Validate(ErrStoragePlaceIsNotConstructed)
}
