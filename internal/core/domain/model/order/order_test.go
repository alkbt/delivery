package order_test

import (
	"testing"

	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/core/domain/model/order"
	"delivery/internal/pkg/errs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrder(t *testing.T) {
	validID := kernel.NewUUID()
	validLocation, _ := kernel.NewLocation(5, 7)
	validVolume := 100

	t.Run("should create valid order with all valid parameters", func(t *testing.T) {
		o, err := order.NewOrder(validID, validLocation, validVolume)

		require.NoError(t, err)
		assert.NotNil(t, o)
		require.NoError(t, o.Validate())
		assert.True(t, o.ID().IsEqual(validID))
		assert.Equal(t, validLocation, o.Location())
		assert.Equal(t, validVolume, o.Volume())
		assert.Equal(t, order.Created, o.Status())
		assert.Nil(t, o.Courier())
	})

	t.Run("should fail with invalid UUID", func(t *testing.T) {
		var invalidID kernel.UUID

		o, err := order.NewOrder(invalidID, validLocation, validVolume)

		require.Error(t, err)
		assert.Nil(t, o)
		assert.Contains(t, err.Error(), "UUID must be created")
	})

	t.Run("should fail with invalid location", func(t *testing.T) {
		var invalidLocation kernel.Location

		o, err := order.NewOrder(validID, invalidLocation, validVolume)

		require.Error(t, err)
		assert.Nil(t, o)
		assert.Contains(t, err.Error(), "location must be created")
	})

	t.Run("should fail with zero volume", func(t *testing.T) {
		o, err := order.NewOrder(validID, validLocation, 0)

		require.Error(t, err)
		assert.Nil(t, o)
		assert.Contains(t, err.Error(), "volume is invalid")
		assert.Contains(t, err.Error(), "0 is not greater than 0")
	})

	t.Run("should fail with negative volume", func(t *testing.T) {
		o, err := order.NewOrder(validID, validLocation, -50)

		require.Error(t, err)
		assert.Nil(t, o)
		assert.Contains(t, err.Error(), "volume is invalid")
		assert.Contains(t, err.Error(), "-50 is not greater than 0")
	})

	t.Run("should handle multiple validation errors", func(t *testing.T) {
		var invalidID kernel.UUID
		var invalidLocation kernel.Location

		o, err := order.NewOrder(invalidID, invalidLocation, -1)

		require.Error(t, err)
		assert.Nil(t, o)
		// Should contain all validation errors joined
		assert.Contains(t, err.Error(), "UUID must be created")
		assert.Contains(t, err.Error(), "location must be created")
		assert.Contains(t, err.Error(), "volume is invalid")
	})

	t.Run("should accept minimum valid volume", func(t *testing.T) {
		o, err := order.NewOrder(validID, validLocation, 1)

		require.NoError(t, err)
		assert.Equal(t, 1, o.Volume())
	})

	t.Run("should accept large volume", func(t *testing.T) {
		largeVolume := 999999
		o, err := order.NewOrder(validID, validLocation, largeVolume)

		require.NoError(t, err)
		assert.Equal(t, largeVolume, o.Volume())
	})
}

func TestOrder_Validate(t *testing.T) {
	validID := kernel.NewUUID()
	validLocation, _ := kernel.NewLocation(5, 7)
	validVolume := 100

	t.Run("should pass validation for properly constructed order", func(t *testing.T) {
		o, _ := order.NewOrder(validID, validLocation, validVolume)

		err := o.Validate()

		require.NoError(t, err)
	})

	t.Run("should fail validation for nil order", func(t *testing.T) {
		var o *order.Order

		err := o.Validate()

		require.Error(t, err)
		assert.Equal(t, order.ErrOrderIsNotConstructed, err)
	})

	t.Run("should fail validation for zero value order", func(t *testing.T) {
		var o order.Order

		err := o.Validate()

		require.Error(t, err)
		assert.Equal(t, order.ErrOrderIsNotConstructed, err)
	})
}

func TestOrder_IsEqual(t *testing.T) {
	id1 := kernel.NewUUID()
	id2 := kernel.NewUUID()
	location1, _ := kernel.NewLocation(5, 7)
	location2, _ := kernel.NewLocation(3, 4)

	t.Run("should return true for orders with same ID", func(t *testing.T) {
		o1, _ := order.NewOrder(id1, location1, 100)
		o2, _ := order.NewOrder(id1, location2, 200) // Different location and volume

		assert.True(t, o1.IsEqual(o2))
		assert.True(t, o2.IsEqual(o1))
	})

	t.Run("should return false for orders with different IDs", func(t *testing.T) {
		o1, _ := order.NewOrder(id1, location1, 100)
		o2, _ := order.NewOrder(id2, location1, 100) // Same location and volume

		assert.False(t, o1.IsEqual(o2))
		assert.False(t, o2.IsEqual(o1))
	})

	t.Run("should return false when comparing with nil", func(t *testing.T) {
		o1, _ := order.NewOrder(id1, location1, 100)

		assert.False(t, o1.IsEqual(nil))
	})

	t.Run("should handle self comparison", func(t *testing.T) {
		o1, _ := order.NewOrder(id1, location1, 100)

		assert.True(t, o1.IsEqual(o1))
	})
}

func TestOrder_Getters(t *testing.T) {
	id := kernel.NewUUID()
	location, _ := kernel.NewLocation(5, 7)
	volume := 100

	o, _ := order.NewOrder(id, location, volume)

	t.Run("should return correct ID", func(t *testing.T) {
		assert.True(t, o.ID().IsEqual(id))
	})

	t.Run("should return correct location", func(t *testing.T) {
		equal, _ := o.Location().IsEqual(location)
		assert.True(t, equal)
	})

	t.Run("should return correct volume", func(t *testing.T) {
		assert.Equal(t, volume, o.Volume())
	})

	t.Run("should return correct initial status", func(t *testing.T) {
		assert.Equal(t, order.Created, o.Status())
	})

	t.Run("should return nil courier initially", func(t *testing.T) {
		assert.Nil(t, o.Courier())
	})
}

func TestOrder_Assign(t *testing.T) {
	validID := kernel.NewUUID()
	validLocation, _ := kernel.NewLocation(5, 7)
	validVolume := 100
	courierID := kernel.NewUUID()

	t.Run("should assign courier to created order", func(t *testing.T) {
		o, _ := order.NewOrder(validID, validLocation, validVolume)

		err := o.Assign(courierID)

		require.NoError(t, err)
		assert.Equal(t, order.Assigned, o.Status())
		assert.NotNil(t, o.Courier())
		assert.True(t, o.Courier().IsEqual(courierID))
	})

	t.Run("should reassign courier to already assigned order", func(t *testing.T) {
		o, _ := order.NewOrder(validID, validLocation, validVolume)
		firstCourier := kernel.NewUUID()
		secondCourier := kernel.NewUUID()

		// First assignment
		err := o.Assign(firstCourier)
		require.NoError(t, err)

		// Reassignment
		err = o.Assign(secondCourier)
		require.NoError(t, err)
		assert.Equal(t, order.Assigned, o.Status())
		assert.True(t, o.Courier().IsEqual(secondCourier))
	})

	t.Run("should fail to assign with invalid courier ID", func(t *testing.T) {
		o, _ := order.NewOrder(validID, validLocation, validVolume)
		var invalidCourierID kernel.UUID

		err := o.Assign(invalidCourierID)

		require.Error(t, err)
		assert.Equal(t, kernel.ErrUUIDIsNotConstructed, err)
		assert.Equal(t, order.Created, o.Status()) // Status unchanged
		assert.Nil(t, o.Courier())                 // Courier unchanged
	})

	t.Run("should fail to assign completed order", func(t *testing.T) {
		o, _ := order.NewOrder(validID, validLocation, validVolume)
		// First assign and complete
		_ = o.Assign(courierID)
		_ = o.Complete()

		newCourierID := kernel.NewUUID()
		err := o.Assign(newCourierID)

		require.Error(t, err)
		assert.IsType(t, &errs.ValueIsInvalidError{}, err)
		assert.Contains(t, err.Error(), "Completed is not a valid status to assign")
		assert.Equal(t, order.Completed, o.Status())   // Status unchanged
		assert.True(t, o.Courier().IsEqual(courierID)) // Original courier preserved
	})
}

func TestOrder_Complete(t *testing.T) {
	validID := kernel.NewUUID()
	validLocation, _ := kernel.NewLocation(5, 7)
	validVolume := 100
	courierID := kernel.NewUUID()

	t.Run("should complete assigned order", func(t *testing.T) {
		o, _ := order.NewOrder(validID, validLocation, validVolume)
		_ = o.Assign(courierID)

		err := o.Complete()

		require.NoError(t, err)
		assert.Equal(t, order.Completed, o.Status())
		assert.True(t, o.Courier().IsEqual(courierID)) // Courier preserved
	})

	t.Run("should fail to complete created order", func(t *testing.T) {
		o, _ := order.NewOrder(validID, validLocation, validVolume)

		err := o.Complete()

		require.Error(t, err)
		assert.IsType(t, &errs.ValueIsInvalidError{}, err)
		assert.Contains(t, err.Error(), "Created is not a valid status to complete")
		assert.Equal(t, order.Created, o.Status()) // Status unchanged
	})

	t.Run("should fail to complete already completed order", func(t *testing.T) {
		o, _ := order.NewOrder(validID, validLocation, validVolume)
		_ = o.Assign(courierID)
		_ = o.Complete()

		err := o.Complete()

		require.Error(t, err)
		assert.IsType(t, &errs.ValueIsInvalidError{}, err)
		assert.Contains(t, err.Error(), "Completed is not a valid status to complete")
		assert.Equal(t, order.Completed, o.Status()) // Status unchanged
	})
}

func TestOrder_FullWorkflow(t *testing.T) {
	t.Run("should follow complete order lifecycle", func(t *testing.T) {
		// Setup
		orderID := kernel.NewUUID()
		location, _ := kernel.NewLocation(5, 7)
		volume := 100
		courierID := kernel.NewUUID()

		// Create order
		o, err := order.NewOrder(orderID, location, volume)
		require.NoError(t, err)
		assert.Equal(t, order.Created, o.Status())
		assert.Nil(t, o.Courier())

		// Assign courier
		err = o.Assign(courierID)
		require.NoError(t, err)
		assert.Equal(t, order.Assigned, o.Status())
		assert.NotNil(t, o.Courier())
		assert.True(t, o.Courier().IsEqual(courierID))

		// Complete order
		err = o.Complete()
		require.NoError(t, err)
		assert.Equal(t, order.Completed, o.Status())
		assert.True(t, o.Courier().IsEqual(courierID))

		// Verify final state
		require.NoError(t, o.Validate())
		assert.True(t, o.ID().IsEqual(orderID))
		equal, _ := o.Location().IsEqual(location)
		assert.True(t, equal)
		assert.Equal(t, volume, o.Volume())
	})

	t.Run("should handle reassignment workflow", func(t *testing.T) {
		// Setup
		orderID := kernel.NewUUID()
		location, _ := kernel.NewLocation(3, 8)
		volume := 250
		firstCourier := kernel.NewUUID()
		secondCourier := kernel.NewUUID()

		// Create and assign to first courier
		o, _ := order.NewOrder(orderID, location, volume)
		_ = o.Assign(firstCourier)
		assert.True(t, o.Courier().IsEqual(firstCourier))

		// Reassign to second courier
		err := o.Assign(secondCourier)
		require.NoError(t, err)
		assert.Equal(t, order.Assigned, o.Status())
		assert.True(t, o.Courier().IsEqual(secondCourier))

		// Complete with second courier
		err = o.Complete()
		require.NoError(t, err)
		assert.Equal(t, order.Completed, o.Status())
		assert.True(t, o.Courier().IsEqual(secondCourier))
	})
}

func TestOrder_EdgeCases(t *testing.T) {
	t.Run("should handle boundary location values", func(t *testing.T) {
		orderID := kernel.NewUUID()
		volume := 1

		// Test with minimum coordinates
		minLocation, _ := kernel.NewLocation(kernel.LocationMinX, kernel.LocationMinY)
		o1, err := order.NewOrder(orderID, minLocation, volume)
		require.NoError(t, err)
		equal, _ := o1.Location().IsEqual(minLocation)
		assert.True(t, equal)

		// Test with maximum coordinates
		maxLocation, _ := kernel.NewLocation(kernel.LocationMaxX, kernel.LocationMaxY)
		o2, err := order.NewOrder(orderID, maxLocation, volume)
		require.NoError(t, err)
		equal, _ = o2.Location().IsEqual(maxLocation)
		assert.True(t, equal)
	})

	t.Run("should handle large volume values", func(t *testing.T) {
		orderID := kernel.NewUUID()
		location, _ := kernel.NewLocation(5, 5)
		largeVolume := 2147483647 // Max int32

		o, err := order.NewOrder(orderID, location, largeVolume)
		require.NoError(t, err)
		assert.Equal(t, largeVolume, o.Volume())
	})

	t.Run("should maintain immutability of returned values", func(t *testing.T) {
		orderID := kernel.NewUUID()
		location, _ := kernel.NewLocation(5, 7)
		volume := 100

		o, _ := order.NewOrder(orderID, location, volume)

		// Get references to internal state
		returnedID := o.ID()
		returnedLocation := o.Location()
		returnedVolume := o.Volume()

		// Modify returned values (this should not affect the order)
		// Note: UUID and Location are value objects, so this tests their immutability
		modifiedID := kernel.NewUUID()
		modifiedLocation, _ := kernel.NewLocation(1, 1)

		// Verify original order state is unchanged
		assert.True(t, o.ID().IsEqual(returnedID))
		assert.False(t, o.ID().IsEqual(modifiedID))
		equal, _ := o.Location().IsEqual(returnedLocation)
		assert.True(t, equal)
		equal, _ = o.Location().IsEqual(modifiedLocation)
		assert.False(t, equal)
		assert.Equal(t, returnedVolume, o.Volume())
	})
}

func TestOrder_ConcurrentSafety(t *testing.T) {
	t.Run("should be safe for concurrent read operations", func(t *testing.T) {
		orderID := kernel.NewUUID()
		location, _ := kernel.NewLocation(5, 7)
		volume := 100
		courierID := kernel.NewUUID()

		o, _ := order.NewOrder(orderID, location, volume)
		_ = o.Assign(courierID)

		// Simulate concurrent reads
		done := make(chan bool, 10)
		for range 10 {
			go func() {
				defer func() { done <- true }()

				// Multiple read operations
				_ = o.ID()
				_ = o.Location()
				_ = o.Volume()
				_ = o.Status()
				_ = o.Courier()
				_ = o.Validate()
			}()
		}

		// Wait for all goroutines to complete
		for range 10 {
			<-done
		}

		// Verify state is still consistent
		require.NoError(t, o.Validate())
		assert.Equal(t, order.Assigned, o.Status())
		assert.True(t, o.Courier().IsEqual(courierID))
	})
}

func TestOrder_ErrorMessages(t *testing.T) {
	t.Run("should provide clear error messages for validation failures", func(t *testing.T) {
		testCases := []struct {
			name     string
			volume   int
			expected string
		}{
			{"zero volume", 0, "0 is not greater than 0"},
			{"negative volume", -1, "-1 is not greater than 0"},
			{"large negative volume", -999, "-999 is not greater than 0"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				orderID := kernel.NewUUID()
				location, _ := kernel.NewLocation(5, 7)

				_, err := order.NewOrder(orderID, location, tc.volume)

				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expected)
			})
		}
	})
}
