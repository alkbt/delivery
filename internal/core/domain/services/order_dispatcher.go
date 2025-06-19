package services

import (
	"errors"
	"math"

	"delivery/internal/core/domain/model/courier"
	"delivery/internal/core/domain/model/order"
)

// ErrCourierNotFound is returned when no suitable courier is available for order dispatch.
// This occurs when either no couriers are provided or none of the provided couriers
// can accommodate the order due to capacity constraints or availability issues.
var ErrCourierNotFound = errors.New("courier not found")

// OrderDispatcher is a domain service responsible for finding and assigning the optimal courier
// for a delivery order based on shortest delivery time.
//
// Key responsibilities:
//   - Validating orders before dispatch
//   - Selecting optimal couriers using time-based optimization
//   - Ensuring atomic order assignment workflow
//
// Business rules:
//   - Orders must be valid before dispatch
//   - Couriers must have available capacity
//   - Selection prioritizes minimum delivery time
//   - Order assignment is atomic
//
// Example usage:
//
//	dispatcher := OrderDispatcher{}
//	order, _ := order.NewOrder(id, location, volume)
//	couriers := []*courier.Courier{courier1, courier2, courier3}
//
//	assignedCourier, err := dispatcher.Dispatch(order, couriers)
//	if errors.Is(err, ErrCourierNotFound) {
//	    // No available couriers for this order
//	    return
//	}
//	if err != nil {
//	    // Handle dispatch failure
//	    return
//	}
//	// Order successfully assigned to assignedCourier
type OrderDispatcher struct{}

// NewOrderDispatcher creates a new OrderDispatcher instance.
//
// Returns:
//   - OrderDispatcher: A new instance ready for order dispatch operations
func NewOrderDispatcher() OrderDispatcher {
	return OrderDispatcher{}
}

// Dispatch finds the optimal courier for a given order and executes the assignment workflow.
//
// Parameters:
//   - order: The order to be dispatched (must be valid)
//   - couriers: Slice of available couriers to consider
//
// Returns:
//   - *courier.Courier: The courier assigned to the order
//   - error: ErrCourierNotFound if no suitable courier exists, or other validation/assignment errors
//
// Selection algorithm:
//   - Validates order and each courier
//   - Checks courier capacity constraints
//   - Selects courier with minimum delivery time
//   - Assigns order to selected courier atomically
func (o OrderDispatcher) Dispatch(order *order.Order, couriers []*courier.Courier) (*courier.Courier, error) {
	if err := order.Validate(); err != nil {
		return nil, err
	}

	if err := order.ValidateAssign(); err != nil {
		return nil, err
	}

	bestCourier, err := o.findBestCourier(order, couriers)
	if err != nil {
		return nil, err
	}

	if err = bestCourier.TakeOrder(order); err != nil {
		return nil, err
	}

	if err = order.Assign(bestCourier.ID()); err != nil {
		return nil, err
	}

	return bestCourier, nil
}

// findBestCourier searches through the provided couriers to find the optimal one for the given order.
//
// Parameters:
//   - order: The order for which to find a courier
//   - couriers: Slice of available couriers to evaluate
//
// Returns:
//   - *courier.Courier: The best courier based on delivery time
//   - error: ErrCourierNotFound if no suitable courier exists, or validation errors
//
// Selection criteria:
//   - Validates courier construction
//   - Checks courier capacity for the order
//   - Optimizes for minimum delivery time
//   - Returns first courier in case of ties
func (o OrderDispatcher) findBestCourier(order *order.Order, couriers []*courier.Courier) (*courier.Courier, error) {
	var (
		bestCourier *courier.Courier
		bestTime    = math.MaxFloat64
	)

	for _, c := range couriers {
		if err := c.Validate(); err != nil {
			return nil, err
		}

		freeCourier, err := c.CanTakeOrder(order)
		if err != nil {
			return nil, err
		}

		if !freeCourier {
			continue
		}

		tm, err := c.CalculateTimeToLocation(order.Location())
		if err != nil {
			return nil, err
		}

		if tm < bestTime {
			bestTime = tm
			bestCourier = c
		}
	}

	if bestCourier == nil {
		return nil, ErrCourierNotFound
	}

	return bestCourier, nil
}
