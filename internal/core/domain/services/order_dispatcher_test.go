package services_test

import (
	"testing"

	"delivery/internal/core/domain/model/courier"
	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/core/domain/model/order"
	"delivery/internal/core/domain/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderDispatcher_Dispatch(t *testing.T) {
	validOrderID := kernel.NewUUID()
	validLocation, _ := kernel.NewLocation(5, 7)
	validOrder, _ := order.NewOrder(validOrderID, validLocation, 5)

	t.Run("should dispatch order to best courier with shortest time", func(t *testing.T) {
		// Create couriers at different distances
		courier1ID := kernel.NewUUID()
		courier1Location, _ := kernel.NewLocation(1, 1) // Distance 10, time 5.0
		courier1, _ := courier.NewCourier(courier1ID, "Alice", 2, courier1Location)

		courier2ID := kernel.NewUUID()
		courier2Location, _ := kernel.NewLocation(3, 3) // Distance 6, time 2.0
		courier2, _ := courier.NewCourier(courier2ID, "Bob", 3, courier2Location)

		courier3ID := kernel.NewUUID()
		courier3Location, _ := kernel.NewLocation(6, 8) // Distance 2, time 1.0
		courier3, _ := courier.NewCourier(courier3ID, "Charlie", 2, courier3Location)

		couriers := []*courier.Courier{courier1, courier2, courier3}
		dispatcher := services.OrderDispatcher{}

		result, err := dispatcher.Dispatch(validOrder, couriers)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsEqual(courier3), "should return courier with shortest time")

		// Verify order is assigned
		assert.Equal(t, order.Assigned, validOrder.Status())
		assert.True(t, validOrder.Courier().IsEqual(courier3.ID()))
	})

	t.Run("should dispatch to only available courier", func(t *testing.T) {
		orderID := kernel.NewUUID()
		orderLocation, _ := kernel.NewLocation(8, 8)
		testOrder, _ := order.NewOrder(orderID, orderLocation, 5)

		courierID := kernel.NewUUID()
		courierLocation, _ := kernel.NewLocation(1, 1)
		availableCourier, _ := courier.NewCourier(courierID, "Solo", 2, courierLocation)

		couriers := []*courier.Courier{availableCourier}
		dispatcher := services.OrderDispatcher{}

		result, err := dispatcher.Dispatch(testOrder, couriers)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsEqual(availableCourier))
		assert.Equal(t, order.Assigned, testOrder.Status())
	})

	t.Run("should return error when no couriers provided", func(t *testing.T) {
		orderID := kernel.NewUUID()
		orderLocation, _ := kernel.NewLocation(8, 8)
		testOrder, _ := order.NewOrder(orderID, orderLocation, 5)

		var couriers []*courier.Courier
		dispatcher := services.OrderDispatcher{}

		result, err := dispatcher.Dispatch(testOrder, couriers)

		require.Error(t, err)
		assert.Nil(t, result)
		require.ErrorIs(t, err, services.ErrCourierNotFound)
		assert.Equal(t, order.Created, testOrder.Status()) // Should remain unchanged
	})

	t.Run("should return error when order is invalid", func(t *testing.T) {
		var invalidOrder *order.Order
		courierID := kernel.NewUUID()
		courierLocation, _ := kernel.NewLocation(1, 1)
		validCourier, _ := courier.NewCourier(courierID, "Test", 2, courierLocation)
		couriers := []*courier.Courier{validCourier}
		dispatcher := services.OrderDispatcher{}

		result, err := dispatcher.Dispatch(invalidOrder, couriers)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, order.ErrOrderIsNotConstructed, err)
	})

	t.Run("should return error when order cannot be assigned", func(t *testing.T) {
		orderID := kernel.NewUUID()
		orderLocation, _ := kernel.NewLocation(5, 7)
		testOrder, _ := order.NewOrder(orderID, orderLocation, 5)

		// Assign and complete the order to make it non-assignable
		courierID := kernel.NewUUID()
		_ = testOrder.Assign(courierID)
		_ = testOrder.Complete()

		// Try to dispatch completed order
		newCourierID := kernel.NewUUID()
		newCourierLocation, _ := kernel.NewLocation(1, 1)
		availableCourier, _ := courier.NewCourier(newCourierID, "Available", 2, newCourierLocation)
		couriers := []*courier.Courier{availableCourier}
		dispatcher := services.OrderDispatcher{}

		result, err := dispatcher.Dispatch(testOrder, couriers)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "Completed is not a valid status to assign")
		assert.Equal(t, order.Completed, testOrder.Status()) // Should remain unchanged
	})

	t.Run("should return error when all couriers are unavailable", func(t *testing.T) {
		orderID := kernel.NewUUID()
		orderLocation, _ := kernel.NewLocation(8, 8)
		testOrder, _ := order.NewOrder(orderID, orderLocation, 5)

		// Create courier with storage that cannot accommodate the order
		courierID := kernel.NewUUID()
		courierLocation, _ := kernel.NewLocation(1, 1)
		unavailableCourier, _ := courier.NewCourier(courierID, "Busy", 2, courierLocation)

		// Fill up the courier's storage with a large order
		largeOrderID := kernel.NewUUID()
		largeOrder, _ := order.NewOrder(largeOrderID, courierLocation, 10)
		_ = unavailableCourier.TakeOrder(largeOrder)

		couriers := []*courier.Courier{unavailableCourier}
		dispatcher := services.OrderDispatcher{}

		result, err := dispatcher.Dispatch(testOrder, couriers)

		require.Error(t, err)
		assert.Nil(t, result)
		require.ErrorIs(t, err, services.ErrCourierNotFound)
		assert.Equal(t, order.Created, testOrder.Status()) // Should remain unchanged
	})

	t.Run("should handle mixed availability scenarios", func(t *testing.T) {
		orderID := kernel.NewUUID()
		orderLocation, _ := kernel.NewLocation(8, 8)
		testOrder, _ := order.NewOrder(orderID, orderLocation, 5)

		// Create one unavailable courier and one available courier
		busyCourierID := kernel.NewUUID()
		busyCourierLocation, _ := kernel.NewLocation(1, 1)
		busyCourier, _ := courier.NewCourier(busyCourierID, "Busy", 2, busyCourierLocation)

		// Fill the busy courier's storage
		blockingOrderID := kernel.NewUUID()
		blockingOrder, _ := order.NewOrder(blockingOrderID, busyCourierLocation, 10)
		_ = busyCourier.TakeOrder(blockingOrder)

		availableCourierID := kernel.NewUUID()
		availableCourierLocation, _ := kernel.NewLocation(10, 10)
		availableCourier, _ := courier.NewCourier(availableCourierID, "Available", 2, availableCourierLocation)

		couriers := []*courier.Courier{busyCourier, availableCourier}
		dispatcher := services.OrderDispatcher{}

		result, err := dispatcher.Dispatch(testOrder, couriers)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsEqual(availableCourier))
		assert.Equal(t, order.Assigned, testOrder.Status())
	})

	t.Run("should handle edge case with zero distance", func(t *testing.T) {
		orderID := kernel.NewUUID()
		orderLocation, _ := kernel.NewLocation(5, 7)
		testOrder, _ := order.NewOrder(orderID, orderLocation, 5)

		// Create courier at exact same location as order
		courierID := kernel.NewUUID()
		sameCourier, _ := courier.NewCourier(courierID, "SameLocation", 1, orderLocation)

		couriers := []*courier.Courier{sameCourier}
		dispatcher := services.OrderDispatcher{}

		result, err := dispatcher.Dispatch(testOrder, couriers)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsEqual(sameCourier))
		assert.Equal(t, order.Assigned, testOrder.Status())
	})
}

func TestOrderDispatcher_ComplexScenarios(t *testing.T) {
	t.Run("should handle mixed courier speeds and distances", func(t *testing.T) {
		orderID := kernel.NewUUID()
		orderLocation, _ := kernel.NewLocation(10, 10)
		testOrder, _ := order.NewOrder(orderID, orderLocation, 5)

		// Courier 1: Close but slow (distance 2, speed 1, time 2.0)
		courier1ID := kernel.NewUUID()
		courier1Location, _ := kernel.NewLocation(8, 10)
		courier1, err := courier.NewCourier(courier1ID, "CloseButSlow", 1, courier1Location)
		require.NoError(t, err)
		require.NotNil(t, courier1)

		// Courier 2: Far but fast (distance 18, speed 11, time ~1.6)
		courier2ID := kernel.NewUUID()
		courier2Location, _ := kernel.NewLocation(1, 1)
		courier2, err := courier.NewCourier(courier2ID, "FarButFast", 11, courier2Location)
		require.NoError(t, err)
		require.NotNil(t, courier2)

		couriers := []*courier.Courier{courier1, courier2}
		dispatcher := services.OrderDispatcher{}

		result, err := dispatcher.Dispatch(testOrder, couriers)

		require.NoError(t, err)
		assert.NotNil(t, result)
		// Courier2 should be faster (time ~1.8 vs 2.0)
		assert.True(t, result.IsEqual(courier2))
		assert.Equal(t, order.Assigned, testOrder.Status())
	})

	t.Run("should handle courier capacity constraints", func(t *testing.T) {
		orderID := kernel.NewUUID()
		orderLocation, _ := kernel.NewLocation(5, 7)
		largeOrder, _ := order.NewOrder(orderID, orderLocation, 15)

		// Courier with default bag (volume 10) - cannot take large order
		courier1ID := kernel.NewUUID()
		courier1Location, _ := kernel.NewLocation(1, 1)
		smallCapacityCourier, _ := courier.NewCourier(courier1ID, "SmallBag", 2, courier1Location)

		// Courier with additional storage - can take large order
		courier2ID := kernel.NewUUID()
		courier2Location, _ := kernel.NewLocation(2, 2)
		largeCapacityCourier, _ := courier.NewCourier(courier2ID, "LargeBag", 2, courier2Location)
		_ = largeCapacityCourier.AddStoragePlace("ExtraBag", 20)

		couriers := []*courier.Courier{smallCapacityCourier, largeCapacityCourier}
		dispatcher := services.OrderDispatcher{}

		result, err := dispatcher.Dispatch(largeOrder, couriers)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsEqual(largeCapacityCourier))
		assert.Equal(t, order.Assigned, largeOrder.Status())
	})
}

func TestOrderDispatcher_EmptyInputs(t *testing.T) {
	t.Run("should handle nil courier slice", func(t *testing.T) {
		orderID := kernel.NewUUID()
		orderLocation, _ := kernel.NewLocation(5, 7)
		testOrder, _ := order.NewOrder(orderID, orderLocation, 5)

		dispatcher := services.OrderDispatcher{}
		result, err := dispatcher.Dispatch(testOrder, nil)

		require.Error(t, err)
		assert.Nil(t, result)
		require.ErrorIs(t, err, services.ErrCourierNotFound)
		assert.Equal(t, order.Created, testOrder.Status())
	})

	t.Run("should handle empty courier slice", func(t *testing.T) {
		orderID := kernel.NewUUID()
		orderLocation, _ := kernel.NewLocation(5, 7)
		testOrder, _ := order.NewOrder(orderID, orderLocation, 5)

		var couriers []*courier.Courier
		dispatcher := services.OrderDispatcher{}
		result, err := dispatcher.Dispatch(testOrder, couriers)

		require.Error(t, err)
		assert.Nil(t, result)
		require.ErrorIs(t, err, services.ErrCourierNotFound)
		assert.Equal(t, order.Created, testOrder.Status())
	})
}

func TestOrderDispatcher_CourierValidation(t *testing.T) {
	t.Run("should return error when courier slice contains nil courier", func(t *testing.T) {
		validOrderID := kernel.NewUUID()
		validLocation, _ := kernel.NewLocation(5, 7)
		validOrder, _ := order.NewOrder(validOrderID, validLocation, 5)

		courierID := kernel.NewUUID()
		courierLocation, _ := kernel.NewLocation(1, 1)
		validCourier, _ := courier.NewCourier(courierID, "Valid", 2, courierLocation)

		// Create slice with valid courier and nil courier
		couriers := []*courier.Courier{validCourier, nil}
		dispatcher := services.OrderDispatcher{}

		result, err := dispatcher.Dispatch(validOrder, couriers)

		require.Error(t, err)
		assert.Nil(t, result)
		require.ErrorIs(t, err, courier.ErrCourierIsNotConstructed)
		assert.Equal(t, order.Created, validOrder.Status()) // Should remain unchanged
	})

	t.Run("should return error when courier slice contains invalid courier", func(t *testing.T) {
		validOrderID := kernel.NewUUID()
		validLocation, _ := kernel.NewLocation(5, 7)
		validOrder, _ := order.NewOrder(validOrderID, validLocation, 5)

		courierID := kernel.NewUUID()
		courierLocation, _ := kernel.NewLocation(1, 1)
		validCourier, _ := courier.NewCourier(courierID, "Valid", 2, courierLocation)

		// Create invalid courier (zero value)
		var invalidCourier courier.Courier

		// Create slice with valid and invalid couriers
		couriers := []*courier.Courier{validCourier, &invalidCourier}
		dispatcher := services.OrderDispatcher{}

		result, err := dispatcher.Dispatch(validOrder, couriers)

		require.Error(t, err)
		assert.Nil(t, result)
		require.ErrorIs(t, err, courier.ErrCourierIsNotConstructed)
		assert.Equal(t, order.Created, validOrder.Status()) // Should remain unchanged
	})

	t.Run("should return error on first invalid courier in slice", func(t *testing.T) {
		validOrderID := kernel.NewUUID()
		validLocation, _ := kernel.NewLocation(5, 7)
		validOrder, _ := order.NewOrder(validOrderID, validLocation, 5)

		// Create invalid courier (zero value) as first element
		var invalidCourier courier.Courier

		courierID := kernel.NewUUID()
		courierLocation, _ := kernel.NewLocation(1, 1)
		validCourier, _ := courier.NewCourier(courierID, "Valid", 2, courierLocation)

		// Invalid courier first, then valid courier
		couriers := []*courier.Courier{&invalidCourier, validCourier}
		dispatcher := services.OrderDispatcher{}

		result, err := dispatcher.Dispatch(validOrder, couriers)

		require.Error(t, err)
		assert.Nil(t, result)
		require.ErrorIs(t, err, courier.ErrCourierIsNotConstructed)
		assert.Equal(t, order.Created, validOrder.Status()) // Should remain unchanged
	})

	t.Run("should succeed when all couriers are valid", func(t *testing.T) {
		validOrderID := kernel.NewUUID()
		validLocation, _ := kernel.NewLocation(5, 7)
		validOrder, _ := order.NewOrder(validOrderID, validLocation, 5)

		courier1ID := kernel.NewUUID()
		courier1Location, _ := kernel.NewLocation(1, 1)
		courier1, _ := courier.NewCourier(courier1ID, "Courier1", 2, courier1Location)

		courier2ID := kernel.NewUUID()
		courier2Location, _ := kernel.NewLocation(3, 3)
		courier2, _ := courier.NewCourier(courier2ID, "Courier2", 3, courier2Location)

		couriers := []*courier.Courier{courier1, courier2}
		dispatcher := services.OrderDispatcher{}

		result, err := dispatcher.Dispatch(validOrder, couriers)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, order.Assigned, validOrder.Status())
	})

	t.Run("should validate each courier individually before processing", func(t *testing.T) {
		validOrderID := kernel.NewUUID()
		validLocation, _ := kernel.NewLocation(5, 7)
		validOrder, _ := order.NewOrder(validOrderID, validLocation, 5)

		// Create multiple valid couriers
		courier1ID := kernel.NewUUID()
		courier1Location, _ := kernel.NewLocation(1, 1)
		courier1, _ := courier.NewCourier(courier1ID, "Courier1", 2, courier1Location)

		courier2ID := kernel.NewUUID()
		courier2Location, _ := kernel.NewLocation(3, 3)
		courier2, _ := courier.NewCourier(courier2ID, "Courier2", 3, courier2Location)

		// Create invalid courier in the middle
		var invalidCourier courier.Courier

		courier3ID := kernel.NewUUID()
		courier3Location, _ := kernel.NewLocation(5, 5)
		courier3, _ := courier.NewCourier(courier3ID, "Courier3", 1, courier3Location)

		// Mix valid and invalid couriers
		couriers := []*courier.Courier{courier1, courier2, &invalidCourier, courier3}
		dispatcher := services.OrderDispatcher{}

		// Should fail on the invalid courier
		result, err := dispatcher.Dispatch(validOrder, couriers)

		require.Error(t, err)
		assert.Nil(t, result)
		require.ErrorIs(t, err, courier.ErrCourierIsNotConstructed)
		assert.Equal(t, order.Created, validOrder.Status()) // Should remain unchanged
	})

	t.Run("should handle slice with only invalid couriers", func(t *testing.T) {
		validOrderID := kernel.NewUUID()
		validLocation, _ := kernel.NewLocation(5, 7)
		validOrder, _ := order.NewOrder(validOrderID, validLocation, 5)

		var invalidCourier1 courier.Courier
		var invalidCourier2 courier.Courier

		couriers := []*courier.Courier{&invalidCourier1, &invalidCourier2}
		dispatcher := services.OrderDispatcher{}

		result, err := dispatcher.Dispatch(validOrder, couriers)

		require.Error(t, err)
		assert.Nil(t, result)
		require.ErrorIs(t, err, courier.ErrCourierIsNotConstructed)
		assert.Equal(t, order.Created, validOrder.Status()) // Should remain unchanged
	})
}

func TestOrderDispatcher_StructMethods(t *testing.T) {
	t.Run("should work with zero value OrderDispatcher", func(t *testing.T) {
		var dispatcher services.OrderDispatcher

		orderID := kernel.NewUUID()
		orderLocation, _ := kernel.NewLocation(5, 7)
		testOrder, _ := order.NewOrder(orderID, orderLocation, 5)

		courierID := kernel.NewUUID()
		courierLocation, _ := kernel.NewLocation(1, 1)
		validCourier, _ := courier.NewCourier(courierID, "Test", 2, courierLocation)
		couriers := []*courier.Courier{validCourier}

		result, err := dispatcher.Dispatch(testOrder, couriers)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsEqual(validCourier))
		assert.Equal(t, order.Assigned, testOrder.Status())
	})

	t.Run("should work with pointer to OrderDispatcher", func(t *testing.T) {
		dispatcher := &services.OrderDispatcher{}

		orderID := kernel.NewUUID()
		orderLocation, _ := kernel.NewLocation(5, 7)
		testOrder, _ := order.NewOrder(orderID, orderLocation, 5)

		courierID := kernel.NewUUID()
		courierLocation, _ := kernel.NewLocation(1, 1)
		validCourier, _ := courier.NewCourier(courierID, "Test", 2, courierLocation)
		couriers := []*courier.Courier{validCourier}

		result, err := dispatcher.Dispatch(testOrder, couriers)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsEqual(validCourier))
		assert.Equal(t, order.Assigned, testOrder.Status())
	})
}
