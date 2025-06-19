package order

import (
	"errors"
	"fmt"

	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/pkg/errs"
)

var (
	// ErrOrderIsNotConstructed is returned when an Order instance was not created through
	// the NewOrder factory method. This ensures all orders are properly validated.
	ErrOrderIsNotConstructed = errors.New("Order must be created via NewOrder constructor")
)

// Order represents a delivery order in the system. It is the aggregate root that manages
// the order lifecycle from creation through assignment to completion.
//
// Order follows these invariants:
//   - Must have a valid unique identifier
//   - Must have a valid delivery location
//   - Volume must be positive (greater than 0)
//   - Status transitions follow defined business rules
//   - Can only be created through NewOrder constructor
//
// The Order struct uses private fields to ensure encapsulation and maintains
// its invariants through validated methods.
type Order struct {
	// id is the unique identifier for the order
	id kernel.UUID

	// courierID is the assigned courier's ID (nil if unassigned)
	courierID *kernel.UUID

	// location is the delivery destination
	location kernel.Location

	// volume represents the order size/weight (must be positive)
	volume int

	// status represents the current state in the order lifecycle
	status Status

	// guard ensures the order was created via NewOrder
	guard kernel.ConstructorGuard
}

// NewOrder creates a new Order instance with validation. This is the only way to create
// a valid Order, ensuring all business invariants are maintained.
//
// Parameters:
//   - id: Unique identifier for the order (must be valid UUID)
//   - location: Delivery location with validated coordinates
//   - volume: Order volume/size (must be positive)
//
// Returns:
//   - *Order: The created order if all validations pass
//   - error: Validation error if any parameter is invalid
//
// Example:
//
//	orderId := kernel.NewUUID()
//	location, _ := kernel.NewLocation(5, 7)
//	order, err := NewOrder(orderId, location, 100)
//	if err != nil {
//	    // Handle validation error
//	}
//
// The constructor validates all inputs and ensures the order is created
// with Created status and no courier assigned.
func NewOrder(id kernel.UUID, location kernel.Location, volume int) (*Order, error) {
	order := &Order{
		status: Created,
		guard:  kernel.NewConstructorGuard(),
	}

	if err := errors.Join(
		order.setID(id),
		order.setLocation(location),
		order.setVolume(volume),
	); err != nil {
		return nil, err
	}

	return order, nil
}

// RestoreOrder reconstructs an Order aggregate from persistent storage.
// Unlike NewOrder which creates orders in Created status, this constructor restores
// an order to its previously persisted state, including status and courier assignment.
//
// This function enables loading complete order aggregates from the database while
// preserving their operational state at the time of persistence. The restored order
// behaves identically to one created through normal domain operations.
//
// Parameters:
//   - id: Unique identifier for the order
//   - location: Delivery destination coordinates
//   - volume: Order size/weight
//   - status: Current order status
//   - courierID: Assigned courier ID (nil if unassigned)
//
// Returns:
//   - *Order: Restored order aggregate
//   - error: Validation error if any parameter is invalid or state is inconsistent
//
// Business Rules:
//   - Order ID must be valid
//   - Location must be valid coordinates
//   - Volume must be positive
//   - Status must be valid enum value
//   - Courier assignment must be consistent with status
//
// Examples:
//
//	// Restore unassigned order
//	order, err := RestoreOrder(id, location, 100, order.Created, nil)
//	if err != nil {
//	    return fmt.Errorf("restoration failed: %w", err)
//	}
//
//	// Restore assigned order
//	order, err := RestoreOrder(id, location, 100, order.Assigned, &courierID)
//	if err != nil {
//	    return fmt.Errorf("restoration failed: %w", err)
//	}
func RestoreOrder(
	id kernel.UUID,
	location kernel.Location,
	volume int,
	status Status,
	courierID *kernel.UUID,
) (*Order, error) {
	order := &Order{
		guard: kernel.NewConstructorGuard(),
	}

	if err := errors.Join(
		order.setID(id),
		order.setLocation(location),
		order.setVolume(volume),
		order.setStatus(status),
		order.setCourierID(courierID),
	); err != nil {
		return nil, err
	}

	if err := order.ValidateSetStatusCourier(); err != nil {
		return nil, err
	}

	return order, nil
}

// Validate ensures the Order instance was properly constructed through NewOrder.
// This prevents bypassing validation by directly instantiating the struct.
//
// Returns:
//   - nil if the order is valid
//   - ErrOrderIsNotConstructed if the order was not created via NewOrder
//
// This method should be called when reconstructing orders from persistence
// to ensure data integrity.
func (o *Order) Validate() error {
	if o == nil {
		return ErrOrderIsNotConstructed
	}

	return o.guard.Validate(ErrOrderIsNotConstructed)
}

// IsEqual compares two orders by their unique identifiers.
// Orders are considered equal if they have the same ID.
//
// Parameters:
//   - other: The order to compare with
//
// Returns:
//   - true if both orders have the same ID
//   - false if other is nil or IDs differ
func (o *Order) IsEqual(other *Order) bool {
	return other != nil && o.id.IsEqual(other.id)
}

// ID returns the order's unique identifier.
func (o *Order) ID() kernel.UUID {
	return o.id
}

// Location returns the delivery location for the order.
func (o *Order) Location() kernel.Location {
	return o.location
}

// Volume returns the order's volume/size.
func (o *Order) Volume() int {
	return o.volume
}

// Status returns the current status of the order.
func (o *Order) Status() Status {
	return o.status
}

// Courier returns the assigned courier's ID.
// Returns nil if no courier is assigned.
func (o *Order) Courier() *kernel.UUID {
	return o.courierID
}

// ValidateAssign checks if the order can be assigned to a courier.
//
// Valid states for assignment:
//   - Created status (initial assignment)
//   - Assigned status (reassignment allowed)
//
// Invalid states for assignment:
//   - Completed status (final state, no further assignments)
//   - Unknown status (invalid state)
//
// Returns:
//   - nil if assignment is allowed
//   - error with details if assignment is not allowed
//
// This method provides assignment validation without side effects,
// useful for pre-validation before courier search and dispatch logic.
//
// Example:
//
//	if err := order.ValidateAssign(); err != nil {
//	    // Handle non-assignable order (e.g., already completed)
//	    return err
//	}
//	// Proceed with courier search and assignment
func (o *Order) ValidateAssign() error {
	return o.status.ValidateAssign()
}

// Assign assigns the order to a courier and updates the status to Assigned.
//
// This method enforces the following business rules:
//   - The courier ID must be valid
//   - The order must be in Created or Assigned status
//   - Reassignment is allowed (from Assigned to Assigned)
//
// Parameters:
//   - courierID: The ID of the courier to assign
//
// Returns:
//   - nil on successful assignment
//   - error if courier ID is invalid or status transition is not allowed
//
// Example:
//
//	courierId := kernel.NewUUID()
//	err := order.Assign(courierId)
//	if err != nil {
//	    // Handle assignment failure
//	}
//
// After successful assignment, the order's status becomes Assigned and
// Courier() will return the assigned courier's ID.
func (o *Order) Assign(courierID kernel.UUID) error {
	if err := courierID.Validate(); err != nil {
		return err
	}

	newStatus, err := o.status.Assign()
	if err != nil {
		return err
	}

	o.status = newStatus
	o.courierID = &courierID
	return nil
}

// Complete marks the order as completed (delivered).
//
// This method enforces the following business rules:
//   - The order must be in Assigned status
//   - Completed is a final state with no further transitions
//
// Returns:
//   - nil on successful completion
//   - error if the order is not in Assigned status
//
// Example:
//
//	err := order.Complete()
//	if err != nil {
//	    // Order was not in Assigned status
//	}
//
// After successful completion, the order's status becomes Completed,
// which is the final state in the order lifecycle.
func (o *Order) Complete() error {
	newStatus, err := o.status.Complete()
	if err != nil {
		return err
	}

	o.status = newStatus
	return nil
}

// setID validates and sets the order's unique identifier.
// This is a private method used only during construction.
func (o *Order) setID(id kernel.UUID) error {
	if err := id.Validate(); err != nil {
		return err
	}
	o.id = id
	return nil
}

// setLocation validates and sets the order's delivery location.
// This is a private method used only during construction.
func (o *Order) setLocation(location kernel.Location) error {
	if err := location.Validate(); err != nil {
		return err
	}
	o.location = location
	return nil
}

// setVolume validates and sets the order's volume.
// Volume must be positive (greater than 0).
// This is a private method used only during construction.
func (o *Order) setVolume(volume int) error {
	if volume <= 0 {
		return errs.NewValueIsInvalidErrorWithCause("volume is invalid", fmt.Errorf("%d is not greater than 0", volume))
	}
	o.volume = volume
	return nil
}

// setStatus sets the order status with validation.
// Used during order restoration to establish the persisted status.
func (o *Order) setStatus(status Status) error {
	if err := status.Validate(); err != nil {
		return err
	}
	o.status = status
	return nil
}

// setCourierID sets the assigned courier ID with validation.
// Used during order restoration to establish courier assignment from persistent state.
func (o *Order) setCourierID(id *kernel.UUID) error {
	if id != nil {
		if err := id.Validate(); err != nil {
			return err
		}
	}

	o.courierID = id
	return nil
}

// ValidateSetStatusCourier validates consistency between order status and courier assignment.
// Used during order restoration to ensure the persisted state is valid.
func (o *Order) ValidateSetStatusCourier() error {
	return o.status.ValidateCanHaveCourier(o.courierID != nil)
}
