package courier_test

import (
	"fmt"
	"testing"

	"delivery/internal/core/domain/model/courier"
	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/pkg/errs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper functions.
func createValidStoragePlace(t *testing.T) *courier.StoragePlace {
	t.Helper()
	place, err := courier.NewStoragePlace(
		kernel.NewUUID(),
		"Test Storage Place",
		1000,
	)
	require.NoError(t, err)
	require.NotNil(t, place)
	return place
}

func createValidOrderID(t *testing.T) kernel.UUID {
	t.Helper()
	return kernel.NewUUID()
}

func TestNewStoragePlace(t *testing.T) {
	validID := kernel.NewUUID()
	validName := "Bag"
	validVolume := 1000

	t.Run("should create storage place with valid parameters", func(t *testing.T) {
		place, err := courier.NewStoragePlace(validID, validName, validVolume)

		require.NoError(t, err)
		assert.NotNil(t, place)
		assert.True(t, place.ID().IsEqual(validID))
		assert.Equal(t, validName, place.Name())
		assert.Equal(t, validVolume, place.TotalVolume())
		assert.Nil(t, place.OrderID())
		require.NoError(t, place.Validate())
	})

	t.Run("should return error for invalid UUID", func(t *testing.T) {
		var invalidID kernel.UUID

		place, err := courier.NewStoragePlace(invalidID, validName, validVolume)

		require.Error(t, err)
		assert.Nil(t, place)
		assert.Contains(t, err.Error(), kernel.ErrUUIDIsNotConstructed.Error())
	})

	t.Run("should return error for empty name", func(t *testing.T) {
		place, err := courier.NewStoragePlace(validID, "", validVolume)

		require.Error(t, err)
		assert.Nil(t, place)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("should return error for zero volume", func(t *testing.T) {
		place, err := courier.NewStoragePlace(validID, validName, 0)

		require.Error(t, err)
		assert.Nil(t, place)
		assert.Contains(t, err.Error(), "totalVolume is invalid")
	})

	t.Run("should return error for negative volume", func(t *testing.T) {
		place, err := courier.NewStoragePlace(validID, validName, -100)

		require.Error(t, err)
		assert.Nil(t, place)
		assert.Contains(t, err.Error(), "totalVolume is invalid")
	})

	t.Run("should return aggregated errors for multiple invalid parameters", func(t *testing.T) {
		var invalidID kernel.UUID

		place, err := courier.NewStoragePlace(invalidID, "", -100)

		require.Error(t, err)
		assert.Nil(t, place)

		// Verify that all three validation errors are included
		assert.Contains(t, err.Error(), kernel.ErrUUIDIsNotConstructed.Error())
		assert.Contains(t, err.Error(), "name is required")
		assert.Contains(t, err.Error(), "totalVolume is invalid")
	})

	t.Run("should handle boundary values", func(t *testing.T) {
		testCases := []struct {
			name        string
			volume      int
			shouldError bool
		}{
			{"minimum valid volume", 1, false},
			{"large valid volume", 1000000, false},
			{"zero volume", 0, true},
			{"negative volume", -1, true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				place, err := courier.NewStoragePlace(validID, validName, tc.volume)

				if tc.shouldError {
					require.Error(t, err)
					assert.Nil(t, place)
				} else {
					require.NoError(t, err)
					assert.NotNil(t, place)
					assert.Equal(t, tc.volume, place.TotalVolume())
				}
			})
		}
	})
}

func TestStoragePlace_IsEqual(t *testing.T) {
	id1 := kernel.NewUUID()
	id2 := kernel.NewUUID()

	t.Run("should return true for storage places with same ID", func(t *testing.T) {
		place1, err := courier.NewStoragePlace(id1, "Place 1", 1000)
		require.NoError(t, err)

		place2, err := courier.NewStoragePlace(id1, "Place 2", 2000) // Different name and volume
		require.NoError(t, err)

		assert.True(t, place1.IsEqual(place2))
		assert.True(t, place2.IsEqual(place1))
	})

	t.Run("should return false for storage places with different IDs", func(t *testing.T) {
		place1, err := courier.NewStoragePlace(id1, "Same Name", 1000)
		require.NoError(t, err)

		place2, err := courier.NewStoragePlace(id2, "Same Name", 1000) // Same attributes, different ID
		require.NoError(t, err)

		assert.False(t, place1.IsEqual(place2))
		assert.False(t, place2.IsEqual(place1))
	})

	t.Run("should return false when comparing with nil", func(t *testing.T) {
		place := createValidStoragePlace(t)

		assert.False(t, place.IsEqual(nil))
	})
}

func TestStoragePlace_Getters(t *testing.T) {
	id := kernel.NewUUID()
	name := "Test Warehouse"
	volume := 1500

	place, err := courier.NewStoragePlace(id, name, volume)
	require.NoError(t, err)

	t.Run("should return correct ID", func(t *testing.T) {
		assert.True(t, place.ID().IsEqual(id))
	})

	t.Run("should return correct name", func(t *testing.T) {
		assert.Equal(t, name, place.Name())
	})

	t.Run("should return correct total volume", func(t *testing.T) {
		assert.Equal(t, volume, place.TotalVolume())
	})

	t.Run("should return nil order ID when empty", func(t *testing.T) {
		assert.Nil(t, place.OrderID())
	})

	t.Run("should return order ID when occupied", func(t *testing.T) {
		orderID := createValidOrderID(t)
		storeErr := place.Store(orderID, 500)
		require.NoError(t, storeErr)

		storedOrderID := place.OrderID()
		require.NotNil(t, storedOrderID)
		assert.True(t, storedOrderID.IsEqual(orderID))
	})
}

func TestStoragePlace_CanStore(t *testing.T) {
	place := createValidStoragePlace(t) // 1000 volume capacity

	t.Run("should return true for valid volume in empty place", func(t *testing.T) {
		canStore, err := place.CanStore(500)

		require.NoError(t, err)
		assert.True(t, canStore)
	})

	t.Run("should return true for volume equal to capacity", func(t *testing.T) {
		canStore, err := place.CanStore(1000)

		require.NoError(t, err)
		assert.True(t, canStore)
	})

	t.Run("should return false for volume exceeding capacity", func(t *testing.T) {
		canStore, err := place.CanStore(1500)

		require.NoError(t, err)
		assert.False(t, canStore)
	})

	t.Run("should return error for zero volume", func(t *testing.T) {
		canStore, err := place.CanStore(0)

		require.Error(t, err)
		assert.False(t, canStore)
		assert.IsType(t, &errs.ValueIsInvalidError{}, err)
		assert.Contains(t, err.Error(), "volume is invalid")
	})

	t.Run("should return error for negative volume", func(t *testing.T) {
		canStore, err := place.CanStore(-100)

		require.Error(t, err)
		assert.False(t, canStore)
		assert.IsType(t, &errs.ValueIsInvalidError{}, err)
	})

	t.Run("should return false when place is occupied", func(t *testing.T) {
		// Store an order first
		orderID := createValidOrderID(t)
		err := place.Store(orderID, 300)
		require.NoError(t, err)

		// Try to check if another order can be stored
		canStore, err := place.CanStore(200)

		require.NoError(t, err)
		assert.False(t, canStore)
	})

	t.Run("boundary value testing", func(t *testing.T) {
		emptyPlace := createValidStoragePlace(t)

		testCases := []struct {
			name        string
			volume      int
			shouldError bool
			expected    bool
		}{
			{"minimum valid volume", 1, false, true},
			{"just under capacity", 999, false, true},
			{"exactly at capacity", 1000, false, true},
			{"just over capacity", 1001, false, false},
			{"zero volume", 0, true, false},
			{"negative volume", -1, true, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				canStore, err := emptyPlace.CanStore(tc.volume)

				if tc.shouldError {
					require.Error(t, err)
					assert.False(t, canStore)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tc.expected, canStore)
				}
			})
		}
	})
}

func TestStoragePlace_Store(t *testing.T) {
	t.Run("should store order successfully with valid parameters", func(t *testing.T) {
		place := createValidStoragePlace(t)
		orderID := createValidOrderID(t)

		err := place.Store(orderID, 500)

		require.NoError(t, err)
		assert.NotNil(t, place.OrderID())
		assert.True(t, place.OrderID().IsEqual(orderID))
	})

	t.Run("should store order with volume equal to capacity", func(t *testing.T) {
		place := createValidStoragePlace(t) // 1000 capacity
		orderID := createValidOrderID(t)

		err := place.Store(orderID, 1000)

		require.NoError(t, err)
		assert.NotNil(t, place.OrderID())
		assert.True(t, place.OrderID().IsEqual(orderID))
	})

	t.Run("should return error for invalid order ID", func(t *testing.T) {
		place := createValidStoragePlace(t)
		var invalidOrderID kernel.UUID

		err := place.Store(invalidOrderID, 500)

		require.Error(t, err)
		assert.Equal(t, kernel.ErrUUIDIsNotConstructed, err)
		assert.Nil(t, place.OrderID())
	})

	t.Run("should return error for invalid volume", func(t *testing.T) {
		place := createValidStoragePlace(t)
		orderID := createValidOrderID(t)

		testCases := []struct {
			name   string
			volume int
		}{
			{"zero volume", 0},
			{"negative volume", -100},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := place.Store(orderID, tc.volume)

				require.Error(t, err)
				assert.IsType(t, &errs.ValueIsInvalidError{}, err)
				assert.Nil(t, place.OrderID())
			})
		}
	})

	t.Run("should return error when volume exceeds capacity", func(t *testing.T) {
		place := createValidStoragePlace(t) // 1000 capacity
		orderID := createValidOrderID(t)

		err := place.Store(orderID, 1500)

		require.Error(t, err)
		assert.Equal(t, courier.ErrCannotStoreOrderInThisStoragePlace, err)
		assert.Nil(t, place.OrderID())
	})

	t.Run("should return error when place is already occupied", func(t *testing.T) {
		place := createValidStoragePlace(t)
		firstOrderID := createValidOrderID(t)
		secondOrderID := createValidOrderID(t)

		// Store first order
		err := place.Store(firstOrderID, 300)
		require.NoError(t, err)

		// Try to store second order
		err = place.Store(secondOrderID, 200)

		require.Error(t, err)
		assert.Equal(t, courier.ErrCannotStoreOrderInThisStoragePlace, err)
		// Verify first order is still stored
		assert.True(t, place.OrderID().IsEqual(firstOrderID))
	})

	t.Run("boundary value testing", func(t *testing.T) {
		testCases := []struct {
			name        string
			volume      int
			shouldError bool
			errorType   error
		}{
			{"minimum valid volume", 1, false, nil},
			{"just under capacity", 999, false, nil},
			{"exactly at capacity", 1000, false, nil},
			{"just over capacity", 1001, true, courier.ErrCannotStoreOrderInThisStoragePlace},
			{"way over capacity", 2000, true, courier.ErrCannotStoreOrderInThisStoragePlace},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				place := createValidStoragePlace(t)
				orderID := createValidOrderID(t)

				err := place.Store(orderID, tc.volume)

				if tc.shouldError {
					require.Error(t, err)
					assert.Equal(t, tc.errorType, err)
					assert.Nil(t, place.OrderID())
				} else {
					require.NoError(t, err)
					assert.NotNil(t, place.OrderID())
					assert.True(t, place.OrderID().IsEqual(orderID))
				}
			})
		}
	})
}

func TestStoragePlace_Clear(t *testing.T) {
	t.Run("should clear stored order successfully", func(t *testing.T) {
		place := createValidStoragePlace(t)
		orderID := createValidOrderID(t)

		// Store order first
		err := place.Store(orderID, 500)
		require.NoError(t, err)
		require.NotNil(t, place.OrderID())

		// Clear the order
		err = place.Clear(orderID)

		require.NoError(t, err)
		assert.Nil(t, place.OrderID())
	})

	t.Run("should return error for invalid order ID", func(t *testing.T) {
		place := createValidStoragePlace(t)
		orderID := createValidOrderID(t)
		var invalidOrderID kernel.UUID

		// Store valid order first
		err := place.Store(orderID, 500)
		require.NoError(t, err)

		// Try to clear with invalid ID
		err = place.Clear(invalidOrderID)

		require.Error(t, err)
		assert.Equal(t, kernel.ErrUUIDIsNotConstructed, err)
		// Verify order is still stored
		assert.NotNil(t, place.OrderID())
		assert.True(t, place.OrderID().IsEqual(orderID))
	})

	t.Run("should return error when place is empty", func(t *testing.T) {
		place := createValidStoragePlace(t)
		orderID := createValidOrderID(t)

		err := place.Clear(orderID)

		require.Error(t, err)
		assert.Equal(t, courier.ErrOrderNotStoredInThisPlace, err)
		assert.Nil(t, place.OrderID())
	})

	t.Run("should return error when wrong order ID is provided", func(t *testing.T) {
		place := createValidStoragePlace(t)
		storedOrderID := createValidOrderID(t)
		differentOrderID := createValidOrderID(t)

		// Store one order
		err := place.Store(storedOrderID, 500)
		require.NoError(t, err)

		// Try to clear with different order ID
		err = place.Clear(differentOrderID)

		require.Error(t, err)
		assert.Equal(t, courier.ErrOrderNotStoredInThisPlace, err)
		// Verify original order is still stored
		assert.NotNil(t, place.OrderID())
		assert.True(t, place.OrderID().IsEqual(storedOrderID))
	})

	t.Run("should allow storing new order after clearing", func(t *testing.T) {
		place := createValidStoragePlace(t)
		firstOrderID := createValidOrderID(t)
		secondOrderID := createValidOrderID(t)

		// Store first order
		err := place.Store(firstOrderID, 300)
		require.NoError(t, err)

		// Clear first order
		err = place.Clear(firstOrderID)
		require.NoError(t, err)

		// Store second order
		err = place.Store(secondOrderID, 700)

		require.NoError(t, err)
		assert.NotNil(t, place.OrderID())
		assert.True(t, place.OrderID().IsEqual(secondOrderID))
	})
}

func TestStoragePlace_Validate(t *testing.T) {
	t.Run("should return nil for properly constructed storage place", func(t *testing.T) {
		place := createValidStoragePlace(t)

		err := place.Validate()

		require.NoError(t, err)
	})

	t.Run("should return error for zero value storage place", func(t *testing.T) {
		var place courier.StoragePlace

		err := place.Validate()

		require.Error(t, err)
		assert.Equal(t, courier.ErrStoragePlaceIsNotConstructed, err)
	})

	t.Run("should return error for nil storage place", func(t *testing.T) {
		var place *courier.StoragePlace

		err := place.Validate()

		require.Error(t, err)
		assert.Equal(t, courier.ErrStoragePlaceIsNotConstructed, err)
	})
}

func TestStoragePlace_ComplexScenarios(t *testing.T) {
	t.Run("complete workflow: store and clear multiple times", func(t *testing.T) {
		place := createValidStoragePlace(t)

		orders := []struct {
			id     kernel.UUID
			volume int
		}{
			{createValidOrderID(t), 300},
			{createValidOrderID(t), 800},
			{createValidOrderID(t), 1000},
		}

		for i, order := range orders {
			t.Run(fmt.Sprintf("iteration_%d", i+1), func(t *testing.T) {
				// Verify place is empty
				assert.Nil(t, place.OrderID())

				// Check if order can be stored
				canStore, err := place.CanStore(order.volume)
				require.NoError(t, err)
				assert.True(t, canStore)

				// Store the order
				err = place.Store(order.id, order.volume)
				require.NoError(t, err)
				assert.True(t, place.OrderID().IsEqual(order.id))

				// Verify cannot store another order
				anotherOrderID := createValidOrderID(t)
				err = place.Store(anotherOrderID, 100)
				require.Error(t, err)
				assert.Equal(t, courier.ErrCannotStoreOrderInThisStoragePlace, err)

				// Clear the order
				err = place.Clear(order.id)
				require.NoError(t, err)
				assert.Nil(t, place.OrderID())
			})
		}
	})

	t.Run("error resilience: operations should not change state on error", func(t *testing.T) {
		place := createValidStoragePlace(t)
		validOrderID := createValidOrderID(t)

		// Store valid order
		err := place.Store(validOrderID, 500)
		require.NoError(t, err)

		// Try invalid operations that should not change state
		var invalidOrderID kernel.UUID

		// Invalid store operation
		err = place.Store(createValidOrderID(t), 200)
		require.Error(t, err)
		assert.True(t, place.OrderID().IsEqual(validOrderID)) // State unchanged

		// Invalid clear operation
		err = place.Clear(invalidOrderID)
		require.Error(t, err)
		assert.True(t, place.OrderID().IsEqual(validOrderID)) // State unchanged

		// Wrong order clear operation
		err = place.Clear(createValidOrderID(t))
		require.Error(t, err)
		assert.True(t, place.OrderID().IsEqual(validOrderID)) // State unchanged
	})
}

func TestStoragePlace_ConcurrentUsage(t *testing.T) {
	t.Run("storage place should be safe for concurrent reads", func(t *testing.T) {
		place := createValidStoragePlace(t)
		orderID := createValidOrderID(t)

		// Store an order
		err := place.Store(orderID, 500)
		require.NoError(t, err)

		// This test verifies that multiple goroutines can safely read
		// the storage place state without data races
		done := make(chan bool, 10)

		for range 10 {
			go func() {
				defer func() { done <- true }()

				// Perform multiple read operations
				assert.True(t, place.ID().IsEqual(place.ID()))
				assert.Equal(t, "Test Storage Place", place.Name())
				assert.Equal(t, 1000, place.TotalVolume())
				assert.NotNil(t, place.OrderID())
				assert.NoError(t, place.Validate())

				canStore, canStoreErr := place.CanStore(200)
				assert.NoError(t, canStoreErr)
				assert.False(t, canStore)
			}()
		}

		// Wait for all goroutines to complete
		for range 10 {
			<-done
		}
	})
}
