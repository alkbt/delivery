package courier

import (
	"errors"

	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/core/domain/model/order"
	"delivery/internal/pkg/errs"
	"delivery/internal/pkg/guard"
)

const (
	// courierDefaultBagName is the default name for the courier's primary storage bag.
	courierDefaultBagName = "Сумка"
	// courierDefaultBagVolume is the default volume capacity for the courier's primary storage bag.
	courierDefaultBagVolume = 10
)

// Domain errors for courier operations.
var (
	// ErrNameIsRequired is returned when attempting to create a courier without a name.
	ErrNameIsRequired = errs.NewValueIsRequiredError("name")
	// ErrSpeedIsRequired is returned when attempting to create a courier with invalid speed (≤0).
	ErrSpeedIsRequired = errs.NewValueIsRequiredError("speed")
	// ErrCourierIsNotConstructed is returned when using an improperly initialized Courier.
	ErrCourierIsNotConstructed = errors.New("Courier must be created via NewCourier constructor")
	// ErrStoragePlaceNotFound is returned when a requested storage place cannot be found.
	ErrStoragePlaceNotFound = errors.New("storage place not found")
)

// Courier represents a delivery courier in the system.
// It is an aggregate root that manages courier identity, movement, and order handling capabilities.
// Couriers can move on the delivery grid, carry orders in storage places, and calculate delivery times.
//
// Key responsibilities:
//   - Managing courier identity (ID, name, speed)
//   - Handling movement on the delivery grid with speed-based constraints
//   - Managing storage places for order transportation
//   - Calculating delivery times based on distance and speed
//   - Validating order capacity before taking orders
//
// Business rules:
//   - Courier must have a valid UUID, non-empty name, and positive speed
//   - Movement is constrained by speed (maximum steps per move operation)
//   - Movement prioritizes X-axis over Y-axis when both directions are needed
//   - Each courier starts with a default storage bag
//   - Orders can only be taken if there's available storage capacity
//
// Example usage:
//
//	location, _ := kernel.NewLocation(1, 1)
//	courier, err := NewCourier(kernel.NewUUID(), "John Doe", 3, location)
//	if err != nil {
//	    // Handle construction error
//	}
//	// Courier is ready to move and handle orders
type Courier struct {
	// id uniquely identifies the courier
	id kernel.UUID
	// name is the human-readable name of the courier
	name string
	// speed determines how many steps the courier can move per turn
	speed int
	// location is the current position of the courier on the delivery grid
	location kernel.Location
	// storagePlaces are the available storage containers for carrying orders
	storagePlaces []*StoragePlace
	// guard ensures the courier was properly constructed
	guard guard.ConstructorGuard
}

// NewCourier creates a new Courier with the specified parameters.
// This is the only way to create a valid Courier instance.
//
// The constructor validates all input parameters and automatically creates a default storage bag.
// All parameters must be valid for the courier to be created successfully.
//
// Parameters:
//   - id: Unique identifier for the courier (must be valid UUID)
//   - name: Human-readable name (must be non-empty)
//   - speed: Movement speed in steps per turn (must be positive)
//   - location: Initial position on the delivery grid (must be valid location)
//
// Returns:
//   - *Courier: A fully initialized courier ready for operations
//   - error: Validation error if any parameter is invalid (aggregated errors for multiple issues)
//
// Business rules applied:
//   - Creates a default storage bag with 10 volume capacity
//   - Validates all input parameters before construction
//   - Uses constructor guard pattern to prevent invalid instances
//
// Example:
//
//	location, _ := kernel.NewLocation(5, 7)
//	courier, err := NewCourier(kernel.NewUUID(), "Alice", 2, location)
//	if err != nil {
//	    log.Fatal("Failed to create courier:", err)
//	}
//	fmt.Printf("Created courier: %s at %s", courier.Name(), courier.Location())
func NewCourier(id kernel.UUID, name string, speed int, location kernel.Location) (*Courier, error) {
	courier := &Courier{
		guard: guard.NewConstructorGuard(),
	}

	if err := errors.Join(
		courier.setID(id),
		courier.setName(name),
		courier.setSpeed(speed),
		courier.setLocation(location),
		courier.AddStoragePlace(courierDefaultBagName, courierDefaultBagVolume),
	); err != nil {
		return nil, err
	}

	return courier, nil
}

// RestoreCourier reconstructs a Courier aggregate from persistent storage.
// Unlike NewCourier which creates fresh couriers with default storage, this constructor
// restores a courier to its previously persisted state, including all storage places
// and their occupancy status.
//
// This function enables loading complete courier aggregates from the database while
// preserving their operational state at the time of persistence. The restored courier
// behaves identically to one created through normal domain operations.
//
// Parameters:
//   - id: Unique identifier for the courier
//   - name: Human-readable courier name
//   - speed: Movement speed in steps per turn
//   - location: Current position on delivery grid
//   - storagePlaces: Collection of storage places belonging to this courier
//
// Returns:
//   - *Courier: Restored courier aggregate
//   - error: Validation error if any parameter is invalid
//
// Business Rules:
//   - Courier ID must be valid
//   - Name cannot be empty
//   - Speed must be positive
//   - Location must be valid coordinates
//   - Must have at least one storage place
//   - All storage places must be valid
//
// Examples:
//
//	// Restore courier with storage places
//	courier, err := RestoreCourier(
//	    courierID,
//	    "Alice Johnson",
//	    3,
//	    location,
//	    storagePlaces,
//	)
//	if err != nil {
//	    return fmt.Errorf("restoration failed: %w", err)
//	}
func RestoreCourier(
	id kernel.UUID,
	name string,
	speed int,
	location kernel.Location,
	storagePlaces []*StoragePlace,
) (*Courier, error) {
	courier := &Courier{
		guard: guard.NewConstructorGuard(),
	}

	if err := errors.Join(
		courier.setID(id),
		courier.setName(name),
		courier.setSpeed(speed),
		courier.setLocation(location),
		courier.setStoragePlaces(storagePlaces),
	); err != nil {
		return nil, err
	}

	return courier, nil
}

// IsEqual compares two couriers for equality based on their unique identifiers.
// Two couriers are considered equal if they have the same ID, regardless of other attributes.
// This method is used for courier identification and deduplication.
//
// Parameters:
//   - other: The courier to compare with (can be nil)
//
// Returns:
//   - bool: true if couriers have the same ID, false otherwise
//
// Example:
//
//	courier1, _ := NewCourier(id, "Alice", 2, location)
//	courier2, _ := NewCourier(id, "Bob", 3, location)  // Same ID, different attributes
//	equal := courier1.IsEqual(courier2)  // true - same ID
func (c *Courier) IsEqual(other *Courier) bool {
	if other == nil {
		return false
	}
	return c.id.IsEqual(other.id)
}

// Validate checks if the Courier was properly constructed using the NewCourier constructor.
// The zero value of Courier is invalid and will fail this validation.
// This method is used internally to ensure courier integrity before operations.
//
// Returns:
//   - error: ErrCourierIsNotConstructed if improperly initialized, nil if valid
//
// Example:
//
//	var courier Courier  // Zero value - invalid
//	if err := courier.Validate(); err != nil {
//	    fmt.Println("Invalid courier:", err)
//	}
func (c *Courier) Validate() error {
	if c == nil {
		return ErrCourierIsNotConstructed
	}
	return c.guard.Validate(ErrCourierIsNotConstructed)
}

// ID returns the unique identifier of the courier.
// The ID is immutable and set during courier construction.
//
// Returns:
//   - kernel.UUID: The courier's unique identifier
//
// Example:
//
//	courier, _ := NewCourier(id, "Alice", 2, location)
//	courierID := courier.ID()  // Returns the same ID used in constructor
func (c *Courier) ID() kernel.UUID {
	return c.id
}

// Name returns the human-readable name of the courier.
// The name is immutable and set during courier construction.
//
// Returns:
//   - string: The courier's name (guaranteed to be non-empty for valid couriers)
//
// Example:
//
//	courier, _ := NewCourier(id, "Alice", 2, location)
//	name := courier.Name()  // "Alice"
func (c *Courier) Name() string {
	return c.name
}

// Speed returns the movement speed of the courier in steps per turn.
// Speed determines how many grid steps the courier can move in a single Move operation.
// The speed is immutable and set during courier construction.
//
// Returns:
//   - int: The courier's speed (guaranteed to be positive for valid couriers)
//
// Example:
//
//	courier, _ := NewCourier(id, "Alice", 3, location)
//	speed := courier.Speed()  // 3
//	// This courier can move up to 3 steps per Move() call
func (c *Courier) Speed() int {
	return c.speed
}

// Location returns the current position of the courier on the delivery grid.
// The location can change through Move operations.
//
// Returns:
//   - kernel.Location: The courier's current position
//
// Example:
//
//	courier, _ := NewCourier(id, "Alice", 2, location)
//	currentPos := courier.Location()
//	fmt.Printf("Courier is at %s", currentPos)
func (c *Courier) Location() kernel.Location {
	return c.location
}

// StoragePlaces returns all storage containers available to the courier.
// Storage places are used to carry orders during delivery.
// The returned slice is a copy to prevent external modification.
//
// Returns:
//   - []*StoragePlace: Array of all storage places (at least one default bag)
//
// Example:
//
//	courier, _ := NewCourier(id, "Alice", 2, location)
//	storage := courier.StoragePlaces()
//	fmt.Printf("Courier has %d storage places", len(storage))  // At least 1
func (c *Courier) StoragePlaces() []*StoragePlace {
	out := make([]*StoragePlace, len(c.storagePlaces))
	copy(out, c.storagePlaces)
	return out
}

// AddStoragePlace creates and adds a new storage container to the courier.
// This allows expanding the courier's carrying capacity by adding additional storage.
// Each storage place has its own capacity and can hold one order at a time.
//
// Parameters:
//   - name: Human-readable name for the storage place (must be non-empty)
//   - volume: Maximum volume capacity (must be positive)
//
// Returns:
//   - error: Validation error if parameters are invalid
//
// Business rules:
//   - Storage place names should be descriptive (e.g., "Backpack", "Side Bag")
//   - Volume determines the maximum order size that can be stored
//   - Each storage place can hold exactly one order
//
// Example:
//
//	courier, _ := NewCourier(id, "Alice", 2, location)
//	err := courier.AddStoragePlace("Backpack", 15)
//	if err != nil {
//	    log.Fatal("Failed to add storage:", err)
//	}
//	// Courier now has additional 15-volume storage capacity
func (c *Courier) AddStoragePlace(name string, volume int) error {
	storagePlace, err := NewStoragePlace(kernel.NewUUID(), name, volume)
	if err != nil {
		return err
	}

	c.storagePlaces = append(c.storagePlaces, storagePlace)
	return nil
}

// CanTakeOrder checks if the courier can accept a specific order.
// This method validates order capacity against available storage without actually taking the order.
// It's used for order assignment decisions and capacity planning.
//
// Parameters:
//   - order: The order to check (must be valid)
//
// Returns:
//   - bool: true if the courier can take the order, false if no capacity
//   - error: Validation error if order is invalid
//
// Business rules:
//   - Order must be valid (proper construction and validation)
//   - At least one storage place must have sufficient free capacity
//   - Order volume must not exceed any individual storage place capacity
//
// Example:
//
//	order, _ := order.NewOrder(id, 5, pickupLoc, deliveryLoc)
//	canTake, err := courier.CanTakeOrder(order)
//	if err != nil {
//	    log.Fatal("Order validation failed:", err)
//	}
//	if canTake {
//	    fmt.Println("Courier can handle this order")
//	}
func (c *Courier) CanTakeOrder(order *order.Order) (bool, error) {
	if err := order.Validate(); err != nil {
		return false, err
	}

	storagePlace, err := c.findStorageForVolume(order.Volume())
	if err != nil {
		return false, err
	}

	return storagePlace != nil, nil
}

// TakeOrder assigns an order to the courier and stores it in available storage.
// This method actually commits the order to the courier's storage, making it unavailable for other orders.
// Use CanTakeOrder first to check capacity before calling this method.
//
// Parameters:
//   - order: The order to take (must be valid and fit in available storage)
//
// Returns:
//   - error: Validation error if order is invalid, or ErrStoragePlaceNotFound if no capacity
//
// Business rules:
//   - Order must be valid and have volume > 0
//   - Must have available storage place with sufficient capacity
//   - Order is stored in the first available storage place that can accommodate it
//   - Once taken, the storage place becomes occupied until order completion
//
// State changes:
//   - Selected storage place becomes occupied with the order
//   - Courier's available capacity is reduced
//
// Example:
//
//	order, _ := order.NewOrder(id, 5, pickupLoc, deliveryLoc)
//	if canTake, _ := courier.CanTakeOrder(order); canTake {
//	    err := courier.TakeOrder(order)
//	    if err != nil {
//	        log.Fatal("Failed to take order:", err)
//	    }
//	    fmt.Println("Order successfully assigned to courier")
//	}
func (c *Courier) TakeOrder(order *order.Order) error {
	if err := order.Validate(); err != nil {
		return err
	}

	storagePlace, err := c.findStorageForVolume(order.Volume())
	if err != nil {
		return err
	}

	if storagePlace == nil {
		return ErrStoragePlaceNotFound
	}

	return storagePlace.Store(order.ID(), order.Volume())
}

// CompleteOrder marks an order as delivered and frees up the associated storage.
// This method should be called when the courier has successfully delivered an order.
// It removes the order from storage, making the storage place available for new orders.
//
// Parameters:
//   - orderID: Unique identifier of the order to complete (must be valid UUID)
//
// Returns:
//   - error: Validation error if orderID is invalid, or ErrStoragePlaceNotFound if order not found
//
// Business rules:
//   - OrderID must be valid and correspond to an order currently carried by the courier
//   - Order must exist in one of the courier's storage places
//   - Completing an order frees up storage capacity immediately
//
// State changes:
//   - Storage place holding the order becomes empty and available
//   - Courier's available capacity increases
//
// Example:
//
//	// After delivering an order
//	err := courier.CompleteOrder(orderID)
//	if err != nil {
//	    log.Fatal("Failed to complete order:", err)
//	}
//	fmt.Println("Order delivered and storage freed")
func (c *Courier) CompleteOrder(orderID kernel.UUID) error {
	if err := orderID.Validate(); err != nil {
		return err
	}

	storagePlace, err := c.findStoragePlaceByOrderID(orderID)
	if err != nil {
		return err
	}

	if storagePlace == nil {
		return ErrStoragePlaceNotFound
	}

	return storagePlace.Clear(orderID)
}

// CalculateTimeToLocation estimates the time required to reach a target location.
// This method calculates the delivery time based on Manhattan distance and courier speed.
// It's used for delivery time estimation and route planning.
//
// Parameters:
//   - target: The destination location (must be valid)
//
// Returns:
//   - float64: Estimated time in turns (distance/speed)
//   - error: Validation error if target location is invalid
//
// Calculation:
//   - Uses Manhattan distance (|x1-x2| + |y1-y2|) between current and target locations
//   - Time = Distance / Speed (in abstract time units/turns)
//   - Returns fractional time for precise estimates
//
// Example:
//
//	targetLocation, _ := kernel.NewLocation(8, 6)
//	time, err := courier.CalculateTimeToLocation(targetLocation)
//	if err != nil {
//	    log.Fatal("Invalid target location:", err)
//	}
//	fmt.Printf("Estimated delivery time: %.2f turns", time)
func (c *Courier) CalculateTimeToLocation(target kernel.Location) (float64, error) {
	if err := target.Validate(); err != nil {
		return 0, err
	}

	distance, err := c.location.Distance(target)
	if err != nil {
		return 0, err
	}

	return float64(distance) / float64(c.speed), nil
}

// Move attempts to move the courier toward a target location.
// This method implements speed-constrained movement with optimized pathfinding.
// The courier moves up to 'speed' steps per call, prioritizing X-axis movement.
//
// Parameters:
//   - target: The destination location (must be valid)
//
// Returns:
//   - error: Validation error if target is invalid, or location setting fails
//
// Movement behavior:
//   - Moves up to 'speed' steps per call (may require multiple calls to reach distant targets)
//   - Uses Manhattan distance pathfinding (no diagonal movement)
//   - Prioritizes X-axis movement: moves horizontally first, then vertically
//   - If already at target location, no movement occurs
//   - Movement is atomic: either succeeds completely or fails without state change
//
// Performance:
//   - Time complexity: O(1) - direct coordinate calculation, no loops
//   - Space complexity: O(1) - uses minimal temporary variables
//
// Example:
//
//	// Courier at (1,1) with speed 3 wants to reach (5,4)
//	target, _ := kernel.NewLocation(5, 4)
//	err := courier.Move(target)
//	if err != nil {
//	    log.Fatal("Movement failed:", err)
//	}
//	// Courier moves to (4,1) - 3 steps horizontally toward target
//	// Call Move again to continue toward target
func (c *Courier) Move(target kernel.Location) error {
	if err := target.Validate(); err != nil {
		return err
	}

	distance, err := c.location.Distance(target)
	if err != nil {
		return err
	}
	if distance == 0 {
		return nil
	}

	steps := minInt(c.speed, distance)
	curX, curY := c.location.X(), c.location.Y()
	tgtX, tgtY := target.X(), target.Y()

	// Move along X axis first
	moveX := minInt(steps, int(abs(tgtX-curX)))
	if curX < tgtX {
		curX += kernel.Coordinate(moveX) //nolint:gosec // it's ok
	} else if curX > tgtX {
		curX -= kernel.Coordinate(moveX) //nolint:gosec  // it's ok
	}
	steps -= moveX

	// Then move along Y axis
	moveY := minInt(steps, int(abs(tgtY-curY)))
	if curY < tgtY {
		curY += kernel.Coordinate(moveY) //nolint:gosec  // it's ok
	} else if curY > tgtY {
		curY -= kernel.Coordinate(moveY) //nolint:gosec  // it's ok
	}

	newLocation, err := kernel.NewLocation(curX, curY)
	if err != nil {
		return err
	}
	return c.setLocation(newLocation)
}

// findStorageForVolume locates the first available storage place that can accommodate the specified volume.
// This is an internal helper method used by order management operations.
// It searches through all storage places and returns the first one with sufficient free capacity.
//
// Parameters:
//   - volume: Required storage volume (must be positive)
//
// Returns:
//   - *StoragePlace: First available storage place with sufficient capacity, or nil if none found
//   - error: Storage validation error if any storage place is in invalid state
//
// Search behavior:
//   - Iterates through storage places in order (first-fit algorithm)
//   - Returns first storage place that can accommodate the volume
//   - Returns nil if no storage place has sufficient capacity
func (c *Courier) findStorageForVolume(volume int) (*StoragePlace, error) {
	for _, storagePlace := range c.storagePlaces {
		canStore, err := storagePlace.CanStore(volume)
		if err != nil {
			return nil, err
		}

		if canStore {
			return storagePlace, nil
		}
	}

	return nil, nil //nolint:nilnil // nothing is found and no error
}

// findStoragePlaceByOrderID locates the storage place containing a specific order.
// This is an internal helper method used to find orders for completion operations.
// It searches through all storage places to find the one holding the specified order.
//
// Parameters:
//   - orderID: Unique identifier of the order to find
//
// Returns:
//   - *StoragePlace: Storage place containing the order, or nil if not found
//   - error: ErrStoragePlaceNotFound if order is not found in any storage place
//
// Search behavior:
//   - Iterates through all storage places
//   - Checks if each occupied storage place contains the target order
//   - Returns the storage place containing the order, or error if not found
func (c *Courier) findStoragePlaceByOrderID(orderID kernel.UUID) (*StoragePlace, error) {
	for _, storagePlace := range c.storagePlaces {
		if storagePlace.OrderID() != nil && storagePlace.OrderID().IsEqual(orderID) {
			return storagePlace, nil
		}
	}

	return nil, ErrStoragePlaceNotFound
}

// setID sets the courier's unique identifier with validation.
// This is an internal setter used during courier construction.
func (c *Courier) setID(id kernel.UUID) error {
	if err := id.Validate(); err != nil {
		return err
	}

	c.id = id
	return nil
}

// setName sets the courier's name with validation.
// This is an internal setter used during courier construction.
func (c *Courier) setName(name string) error {
	if name == "" {
		return ErrNameIsRequired
	}

	c.name = name
	return nil
}

// setSpeed sets the courier's movement speed with validation.
// This is an internal setter used during courier construction.
func (c *Courier) setSpeed(speed int) error {
	if speed <= 0 {
		return ErrSpeedIsRequired
	}

	c.speed = speed
	return nil
}

// setLocation sets the courier's current location with validation.
// This is an internal setter used during construction and movement operations.
func (c *Courier) setLocation(location kernel.Location) error {
	if err := location.Validate(); err != nil {
		return err
	}

	c.location = location
	return nil
}

// setStoragePlaces sets the courier's storage places collection.
// Used during courier restoration to establish the storage places from persistent state.
// Validates that the collection is not empty and all storage places are valid.
func (c *Courier) setStoragePlaces(storagePlaces []*StoragePlace) error {
	if len(storagePlaces) == 0 {
		return errs.NewValueIsRequiredError("storage places are required")
	}

	for _, sp := range storagePlaces {
		if err := sp.Validate(); err != nil {
			return err
		}
	}

	c.storagePlaces = make([]*StoragePlace, len(storagePlaces))
	copy(c.storagePlaces, storagePlaces)
	return nil
}

// minInt returns the smaller of two integers.
// This is a utility function used in movement calculations.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// abs returns the absolute value of a coordinate.
// This is a utility function used in movement distance calculations.
func abs(x kernel.Coordinate) kernel.Coordinate {
	if x < 0 {
		return -x
	}
	return x
}
