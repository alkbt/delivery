package order

import (
	"fmt"

	"delivery/internal/pkg/errs"
)

// Status represents the lifecycle state of an order.
// It implements a state machine with defined transitions to ensure
// orders follow the correct business workflow.
//
// State transitions:
//
//	Created ──┬──> Assigned ──> Completed
//	          │        │
//	          └────────┘
//	     (reassignment allowed)
//
// Status is a value object that validates state transitions
// and provides string representations for persistence and display.
type Status int

const (
	// Unknown represents an invalid or undefined status.
	// This value (0) helps catch uninitialized Status values.
	Unknown Status = iota

	// Created is the initial status when an order is first created.
	// Orders in this status are waiting to be assigned to a courier.
	Created

	// Assigned indicates the order has been assigned to a courier.
	// Orders can be reassigned while in this status.
	Assigned

	// Completed indicates the order has been successfully delivered.
	// This is a final state with no further transitions allowed.
	Completed
)

// getStatusStrings returns a map of Status values to their string representations.
// All statuses are included for string conversion.
func getStatusStrings() map[Status]string {
	return map[Status]string{
		Unknown:   "Unknown",
		Created:   "Created",
		Assigned:  "Assigned",
		Completed: "Completed",
	}
}

// getValidStatusStrings returns a map of only valid Status values.
// Only valid statuses are included to support validation.
func getValidStatusStrings() map[Status]string {
	//nolint:exhaustive // Unknown is intentionally excluded as it's invalid
	return map[Status]string{
		Created:   "Created",
		Assigned:  "Assigned",
		Completed: "Completed",
	}
}

// Validate checks if the Status value is valid.
//
// Valid statuses are: Created, Assigned, Completed.
// Unknown (0) and any other values are invalid.
//
// Returns:
//   - nil if the status is valid
//   - error with details if the status is invalid
//
// This method is used to ensure Status values from external sources
// (e.g., database, API) are valid before use.
func (s Status) Validate() error {
	if _, ok := getValidStatusStrings()[s]; !ok {
		return errs.NewValueIsInvalidErrorWithCause("status is invalid", fmt.Errorf("%d is not a valid status", s))
	}
	return nil
}

// String returns the human-readable name of the status.
//
// Returns:
//   - "Created", "Assigned", or "Completed" for valid statuses
//   - "Unknown" for invalid status values
//
// This method implements the fmt.Stringer interface and is safe
// to call on any Status value, including invalid ones.
//
// Example:
//
//	fmt.Println(order.Status()) // Output: "Assigned"
func (s Status) String() string {
	if str, ok := getStatusStrings()[s]; ok {
		return str
	}
	return "Unknown"
}

// ValidateAssign checks if the status allows assignment without performing the transition.
//
// Valid statuses for assignment:
//   - Created (can be initially assigned)
//   - Assigned (can be reassigned)
//
// Invalid statuses for assignment:
//   - Completed (cannot assign completed orders)
//   - Unknown (invalid status)
//
// Returns:
//   - nil if assignment is allowed from current status
//   - error with details if assignment is not allowed
//
// This method provides assignability validation without side effects,
// useful for pre-validation and business logic checks.
//
// Example:
//
//	if err := status.ValidateAssign(); err != nil {
//	    // Handle non-assignable status
//	    return err
//	}
//	// Proceed with assignment
func (s Status) ValidateAssign() error {
	if s != Created && s != Assigned {
		return errs.NewValueIsInvalidErrorWithCause(
			"status is invalid",
			fmt.Errorf("%s is not a valid status to assign", s.String()),
		)
	}
	return nil
}

// ValidateCanHaveCourier validates the consistency between order status and courier assignment.
// Enforces business rules about which statuses require courier assignment.
//
// Business Rules:
//   - Created orders must not have a courier assigned
//   - Assigned orders must have a courier assigned
//   - Completed orders must have a courier assigned
//
// Parameters:
//   - courier: whether the order has a courier assigned
//
// Returns:
//   - error: validation error if status and courier assignment are inconsistent
func (s Status) ValidateCanHaveCourier(courier bool) error {
	if courier && s != Assigned && s != Completed {
		return errs.NewValueIsInvalidErrorWithCause(
			"status is invalid",
			fmt.Errorf("%s is not a valid status to have a courier", s.String()),
		)
	}

	if !courier && (s == Assigned || s == Completed) {
		return errs.NewValueIsInvalidErrorWithCause(
			"status is invalid",
			fmt.Errorf("%s is not a valid status to have no courier", s.String()),
		)
	}

	return nil
}

// Assign transitions the status to Assigned.
//
// Valid transitions:
//   - Created -> Assigned (initial assignment)
//   - Assigned -> Assigned (reassignment to different courier)
//
// Invalid transitions:
//   - Completed -> Assigned (cannot assign completed orders)
//   - Unknown -> Assigned (invalid initial state)
//
// Returns:
//   - (Assigned, nil) on valid transition
//   - (0, error) if transition is not allowed from current status
//
// This method is used by Order.Assign() to enforce state transitions.
//
// Example:
//
//	newStatus, err := currentStatus.Assign()
//	if err != nil {
//	    // Handle invalid transition
//	}
func (s Status) Assign() (Status, error) {
	if err := s.ValidateAssign(); err != nil {
		return 0, err
	}

	return Assigned, nil
}

// Complete transitions the status to Completed.
//
// Valid transitions:
//   - Assigned -> Completed (order delivered)
//
// Invalid transitions:
//   - Created -> Completed (must be assigned first)
//   - Completed -> Completed (already completed)
//   - Unknown -> Completed (invalid initial state)
//
// Returns:
//   - (Completed, nil) on valid transition
//   - (0, error) if transition is not allowed from current status
//
// This method is used by Order.Complete() to enforce state transitions.
// Completed is a final state with no further transitions possible.
//
// Example:
//
//	newStatus, err := currentStatus.Complete()
//	if err != nil {
//	    // Order was not in Assigned status
//	}
func (s Status) Complete() (Status, error) {
	if s != Assigned {
		return 0, errs.NewValueIsInvalidErrorWithCause(
			"status is invalid",
			fmt.Errorf("%s is not a valid status to complete", s.String()),
		)
	}

	return Completed, nil
}
