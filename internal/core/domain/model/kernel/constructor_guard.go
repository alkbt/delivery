package kernel

import "errors"

// ErrDefaultConstructorGuard is the default error returned by ConstructorGuard.Validate()
// when a nil error is passed as the validation error. This ensures that validation
// always fails with a meaningful message even if no specific error is provided.
var ErrDefaultConstructorGuard = errors.New("object must be created via its constructor")

// ConstructorGuard is a defensive programming pattern that ensures value objects
// and entities are only created through their designated constructor functions.
// It prevents direct struct initialization and enforces validation rules.
//
// This pattern is particularly useful in Domain-Driven Design to maintain invariants
// and ensure that domain objects are always in a valid state. By embedding a
// ConstructorGuard in a struct, you can detect whether the struct was properly
// initialized through its constructor or created as a zero value.
//
// The guard works by maintaining an internal flag that is only set to true when
// the object is created through the proper constructor function. Any attempt to
// use a zero-value struct will fail validation.
//
// Example usage:
//
//	var ErrMoneyNotConstructed = errors.New("Money must be created via NewMoney")
//
//	type Money struct {
//	    amount int
//	    currency string
//	    guard ConstructorGuard
//	}
//
//	func NewMoney(amount int, currency string) (Money, error) {
//	    if amount < 0 {
//	        return Money{}, errors.New("amount cannot be negative")
//	    }
//	    if currency == "" {
//	        return Money{}, errors.New("currency is required")
//	    }
//	    return Money{
//	        amount: amount,
//	        currency: currency,
//	        guard: NewConstructorGuard(),
//	    }, nil
//	}
//
//	func (m Money) Validate() error {
//	    return m.guard.Validate(ErrMoneyNotConstructed)
//	}
//
// Benefits:
//   - Prevents accidental use of zero values
//   - Enforces constructor usage for proper initialization
//   - Maintains domain invariants
//   - Provides clear error messages for invalid construction
//   - Lightweight and has minimal performance overhead
type ConstructorGuard struct {
	isConstructed bool
}

// NewConstructorGuard creates a new ConstructorGuard that marks an object as
// properly constructed. This should be called in the constructor of domain objects
// to ensure they can be distinguished from zero-value instances.
//
// Example:
//
//	func NewOrder(id UUID, location Location) Order {
//	    return Order{
//	        id:       id,
//	        location: location,
//	        guard:    NewConstructorGuard(),
//	    }
//	}
//
// Returns:
//   - A ConstructorGuard with isConstructed set to true
func NewConstructorGuard() ConstructorGuard {
	return ConstructorGuard{isConstructed: true}
}

// Validate checks whether the guarded object was properly constructed through
// its designated constructor function.
//
// If the object was created as a zero value (not through the constructor),
// this method returns the provided validation error. If validationError is nil,
// ErrDefaultConstructorGuard is returned instead.
//
// This method should be called in the Validate method of the domain object
// to ensure proper construction before any operations are performed on it.
//
// Parameters:
//   - validationError: The error to return if the object was not properly constructed
//
// Example:
//
//	var ErrOrderNotConstructed = errors.New("Order must be created via NewOrder")
//
//	func (o Order) Validate() error {
//	    if err := o.guard.Validate(ErrOrderNotConstructed); err != nil {
//	        return err
//	    }
//	    // Additional validation logic...
//	    return nil
//	}
//
// Returns:
//   - nil if the object was properly constructed
//   - validationError if the object was not constructed through its constructor
//   - ErrDefaultConstructorGuard if validationError is nil and object not constructed
func (g ConstructorGuard) Validate(validationError error) error {
	if validationError == nil {
		validationError = ErrDefaultConstructorGuard
	}
	if !g.isConstructed {
		return validationError
	}
	return nil
}
