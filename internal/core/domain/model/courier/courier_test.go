package courier_test

import (
	"fmt"
	"testing"

	"delivery/internal/core/domain/model/courier"
	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/core/domain/model/order"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper functions.
func createValidCourier(t *testing.T) *courier.Courier {
	t.Helper()
	id := kernel.NewUUID()
	location, err := kernel.NewLocation(1, 1)
	require.NoError(t, err)

	c, err := courier.NewCourier(id, "Test Courier", 3, location)
	require.NoError(t, err)
	require.NotNil(t, c)
	return c
}

func createValidLocation(t *testing.T, x, y kernel.Coordinate) kernel.Location {
	t.Helper()
	location, err := kernel.NewLocation(x, y)
	require.NoError(t, err)
	return location
}

func createValidOrder(t *testing.T, volume int) *order.Order {
	t.Helper()
	id := kernel.NewUUID()
	location := createValidLocation(t, 5, 5)

	o, err := order.NewOrder(id, location, volume)
	require.NoError(t, err)
	require.NotNil(t, o)
	return o
}

func TestNewCourier(t *testing.T) {
	validID := kernel.NewUUID()
	validName := "Alice"
	validSpeed := 3
	validLocation := createValidLocation(t, 5, 7)

	t.Run("should create courier with valid parameters", func(t *testing.T) {
		c, err := courier.NewCourier(validID, validName, validSpeed, validLocation)

		require.NoError(t, err)
		assert.NotNil(t, c)
		require.NoError(t, c.Validate())
		assert.True(t, c.ID().IsEqual(validID))
		assert.Equal(t, validName, c.Name())
		assert.Equal(t, validSpeed, c.Speed())
		assert.Equal(t, validLocation, c.Location())

		// Should have default storage bag
		storagePlaces := c.StoragePlaces()
		assert.Len(t, storagePlaces, 1)
		assert.Equal(t, "Сумка", storagePlaces[0].Name())
		assert.Equal(t, 10, storagePlaces[0].TotalVolume())
	})

	t.Run("should return error for invalid UUID", func(t *testing.T) {
		var invalidID kernel.UUID

		c, err := courier.NewCourier(invalidID, validName, validSpeed, validLocation)

		require.Error(t, err)
		assert.Nil(t, c)
		assert.Contains(t, err.Error(), kernel.ErrUUIDIsNotConstructed.Error())
	})

	t.Run("should return error for empty name", func(t *testing.T) {
		c, err := courier.NewCourier(validID, "", validSpeed, validLocation)

		require.Error(t, err)
		assert.Nil(t, c)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("should return error for invalid speed", func(t *testing.T) {
		testCases := []struct {
			name  string
			speed int
		}{
			{"zero speed", 0},
			{"negative speed", -1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				c, err := courier.NewCourier(validID, validName, tc.speed, validLocation)

				require.Error(t, err)
				assert.Nil(t, c)
				assert.Contains(t, err.Error(), "speed")
			})
		}
	})

	t.Run("should return error for invalid location", func(t *testing.T) {
		var invalidLocation kernel.Location

		c, err := courier.NewCourier(validID, validName, validSpeed, invalidLocation)

		require.Error(t, err)
		assert.Nil(t, c)
		assert.Contains(t, err.Error(), "location must be created")
	})

	t.Run("should return aggregated errors for multiple invalid parameters", func(t *testing.T) {
		var invalidID kernel.UUID
		var invalidLocation kernel.Location

		c, err := courier.NewCourier(invalidID, "", -1, invalidLocation)

		require.Error(t, err)
		assert.Nil(t, c)

		// Verify that all validation errors are included
		errorStr := err.Error()
		assert.Contains(t, errorStr, kernel.ErrUUIDIsNotConstructed.Error())
		assert.Contains(t, errorStr, "name")
		assert.Contains(t, errorStr, "speed")
		assert.Contains(t, errorStr, "location must be created")
	})

	t.Run("should handle boundary values", func(t *testing.T) {
		testCases := []struct {
			name        string
			speed       int
			shouldError bool
		}{
			{"minimum valid speed", 1, false},
			{"typical speed", 5, false},
			{"high speed", 100, false},
			{"zero speed", 0, true},
			{"negative speed", -1, true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				c, err := courier.NewCourier(validID, validName, tc.speed, validLocation)

				if tc.shouldError {
					require.Error(t, err)
					assert.Nil(t, c)
				} else {
					require.NoError(t, err)
					assert.NotNil(t, c)
					assert.Equal(t, tc.speed, c.Speed())
				}
			})
		}
	})

	t.Run("should create courier at different grid locations", func(t *testing.T) {
		testCases := []struct {
			name string
			x, y kernel.Coordinate
		}{
			{"top-left corner", 1, 1},
			{"center", 5, 5},
			{"bottom-right corner", 10, 10},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				location := createValidLocation(t, tc.x, tc.y)
				c, err := courier.NewCourier(validID, validName, validSpeed, location)

				require.NoError(t, err)
				assert.NotNil(t, c)
				assert.Equal(t, location, c.Location())
			})
		}
	})

	t.Run("should create couriers with different names", func(t *testing.T) {
		names := []string{
			"Alice",
			"Bob",
			"Courier-123",
			"Очень длинное имя курьера",
			"A", // Single character
		}

		for _, name := range names {
			t.Run(fmt.Sprintf("name_%s", name), func(t *testing.T) {
				c, err := courier.NewCourier(validID, name, validSpeed, validLocation)

				require.NoError(t, err)
				assert.NotNil(t, c)
				assert.Equal(t, name, c.Name())
			})
		}
	})
}

func TestCourier_IsEqual(t *testing.T) {
	id1 := kernel.NewUUID()
	id2 := kernel.NewUUID()
	location := createValidLocation(t, 5, 5)

	t.Run("should return true for couriers with same ID", func(t *testing.T) {
		courier1, err := courier.NewCourier(id1, "Alice", 2, location)
		require.NoError(t, err)

		courier2, err := courier.NewCourier(id1, "Bob", 3, location) // Different name and speed
		require.NoError(t, err)

		assert.True(t, courier1.IsEqual(courier2))
		assert.True(t, courier2.IsEqual(courier1))
	})

	t.Run("should return false for couriers with different IDs", func(t *testing.T) {
		courier1, err := courier.NewCourier(id1, "Same Name", 2, location)
		require.NoError(t, err)

		courier2, err := courier.NewCourier(id2, "Same Name", 2, location) // Same attributes, different ID
		require.NoError(t, err)

		assert.False(t, courier1.IsEqual(courier2))
		assert.False(t, courier2.IsEqual(courier1))
	})

	t.Run("should return false when comparing with nil", func(t *testing.T) {
		c := createValidCourier(t)

		assert.False(t, c.IsEqual(nil))
	})
}

func TestCourier_Validate(t *testing.T) {
	t.Run("should return nil for properly constructed courier", func(t *testing.T) {
		c := createValidCourier(t)

		err := c.Validate()

		require.NoError(t, err)
	})

	t.Run("should return error for zero value courier", func(t *testing.T) {
		var c courier.Courier

		err := c.Validate()

		require.Error(t, err)
		assert.Equal(t, courier.ErrCourierIsNotConstructed, err)
	})

	t.Run("should return error for nil courier", func(t *testing.T) {
		var c *courier.Courier

		err := c.Validate()

		require.Error(t, err)
		assert.Equal(t, courier.ErrCourierIsNotConstructed, err)
	})
}

func TestCourier_Getters(t *testing.T) {
	id := kernel.NewUUID()
	name := "Test Courier"
	speed := 5
	location := createValidLocation(t, 3, 7)

	c, err := courier.NewCourier(id, name, speed, location)
	require.NoError(t, err)

	t.Run("should return correct ID", func(t *testing.T) {
		assert.True(t, c.ID().IsEqual(id))
	})

	t.Run("should return correct name", func(t *testing.T) {
		assert.Equal(t, name, c.Name())
	})

	t.Run("should return correct speed", func(t *testing.T) {
		assert.Equal(t, speed, c.Speed())
	})

	t.Run("should return correct location", func(t *testing.T) {
		assert.Equal(t, location, c.Location())
	})

	t.Run("should return storage places", func(t *testing.T) {
		storagePlaces := c.StoragePlaces()

		assert.NotNil(t, storagePlaces)
		assert.Len(t, storagePlaces, 1) // Default bag
		assert.Equal(t, "Сумка", storagePlaces[0].Name())
		assert.Equal(t, 10, storagePlaces[0].TotalVolume())
	})

	t.Run("should return immutable storage places slice", func(t *testing.T) {
		storagePlaces1 := c.StoragePlaces()
		storagePlaces2 := c.StoragePlaces()

		// Should return different slice instances (defensive copy)
		assert.NotSame(t, &storagePlaces1, &storagePlaces2)
		assert.Len(t, storagePlaces2, len(storagePlaces1))
	})
}

func TestCourier_Move(t *testing.T) {
	t.Run("should not move when already at target location", func(t *testing.T) {
		startLocation := createValidLocation(t, 5, 5)
		c, err := courier.NewCourier(kernel.NewUUID(), "Test", 3, startLocation)
		require.NoError(t, err)

		targetLocation := createValidLocation(t, 5, 5) // Same location

		err = c.Move(targetLocation)

		require.NoError(t, err)
		assert.Equal(t, startLocation, c.Location())
	})

	t.Run("should return error for invalid target location", func(t *testing.T) {
		c := createValidCourier(t)
		var invalidLocation kernel.Location

		err := c.Move(invalidLocation)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "location must be created")
	})

	t.Run("should move within speed limit", func(t *testing.T) {
		startLocation := createValidLocation(t, 1, 1)
		c, err := courier.NewCourier(kernel.NewUUID(), "Test", 3, startLocation)
		require.NoError(t, err)

		targetLocation := createValidLocation(t, 3, 1) // 2 steps horizontally

		err = c.Move(targetLocation)

		require.NoError(t, err)
		expectedLocation := createValidLocation(t, 3, 1) // Should reach target
		assert.Equal(t, expectedLocation, c.Location())
	})

	t.Run("should move maximum speed steps when target is far", func(t *testing.T) {
		startLocation := createValidLocation(t, 1, 1)
		c, err := courier.NewCourier(kernel.NewUUID(), "Test", 2, startLocation) // Speed 2
		require.NoError(t, err)

		targetLocation := createValidLocation(t, 6, 1) // 5 steps away

		err = c.Move(targetLocation)

		require.NoError(t, err)
		expectedLocation := createValidLocation(t, 3, 1) // Should move 2 steps
		assert.Equal(t, expectedLocation, c.Location())
	})

	t.Run("should prioritize X-axis movement", func(t *testing.T) {
		startLocation := createValidLocation(t, 1, 1)
		c, err := courier.NewCourier(kernel.NewUUID(), "Test", 3, startLocation)
		require.NoError(t, err)

		targetLocation := createValidLocation(t, 4, 4) // 3 steps X, 3 steps Y

		err = c.Move(targetLocation)

		require.NoError(t, err)
		expectedLocation := createValidLocation(t, 4, 1) // Should move X first
		assert.Equal(t, expectedLocation, c.Location())
	})

	t.Run("should move Y-axis after X-axis is aligned", func(t *testing.T) {
		startLocation := createValidLocation(t, 5, 1)
		c, err := courier.NewCourier(kernel.NewUUID(), "Test", 3, startLocation)
		require.NoError(t, err)

		targetLocation := createValidLocation(t, 5, 4) // Only Y movement needed

		err = c.Move(targetLocation)

		require.NoError(t, err)
		expectedLocation := createValidLocation(t, 5, 4) // Should reach target (3 steps Y)
		assert.Equal(t, expectedLocation, c.Location())
	})

	t.Run("should handle partial X and Y movement", func(t *testing.T) {
		startLocation := createValidLocation(t, 1, 1)
		c, err := courier.NewCourier(kernel.NewUUID(), "Test", 4, startLocation)
		require.NoError(t, err)

		targetLocation := createValidLocation(t, 3, 4) // 2 steps X, 3 steps Y

		err = c.Move(targetLocation)

		require.NoError(t, err)
		expectedLocation := createValidLocation(t, 3, 3) // 2 steps X, 2 steps Y
		assert.Equal(t, expectedLocation, c.Location())
	})

	t.Run("should handle negative X movement", func(t *testing.T) {
		startLocation := createValidLocation(t, 5, 5)
		c, err := courier.NewCourier(kernel.NewUUID(), "Test", 2, startLocation)
		require.NoError(t, err)

		targetLocation := createValidLocation(t, 2, 5) // Move left 3 steps

		err = c.Move(targetLocation)

		require.NoError(t, err)
		expectedLocation := createValidLocation(t, 3, 5) // Should move 2 steps left
		assert.Equal(t, expectedLocation, c.Location())
	})

	t.Run("should handle negative Y movement", func(t *testing.T) {
		startLocation := createValidLocation(t, 5, 8)
		c, err := courier.NewCourier(kernel.NewUUID(), "Test", 3, startLocation)
		require.NoError(t, err)

		targetLocation := createValidLocation(t, 5, 4) // Move down 4 steps

		err = c.Move(targetLocation)

		require.NoError(t, err)
		expectedLocation := createValidLocation(t, 5, 5) // Should move 3 steps down
		assert.Equal(t, expectedLocation, c.Location())
	})

	t.Run("should handle diagonal movement efficiently", func(t *testing.T) {
		testCases := []struct {
			name                 string
			startX, startY       kernel.Coordinate
			targetX, targetY     kernel.Coordinate
			speed                int
			expectedX, expectedY kernel.Coordinate
		}{
			{
				name:   "northeast movement",
				startX: 1, startY: 1,
				targetX: 4, targetY: 4,
				speed:     5,
				expectedX: 4, expectedY: 3, // 3 steps X, 2 steps Y
			},
			{
				name:   "southwest movement",
				startX: 8, startY: 8,
				targetX: 5, targetY: 5,
				speed:     4,
				expectedX: 5, expectedY: 7, // 3 steps X, 1 step Y
			},
			{
				name:   "southeast movement",
				startX: 2, startY: 8,
				targetX: 6, targetY: 4,
				speed:     6,
				expectedX: 6, expectedY: 6, // 4 steps X, 2 steps Y
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				startLocation := createValidLocation(t, tc.startX, tc.startY)
				c, err := courier.NewCourier(kernel.NewUUID(), "Test", tc.speed, startLocation)
				require.NoError(t, err)

				targetLocation := createValidLocation(t, tc.targetX, tc.targetY)

				err = c.Move(targetLocation)

				require.NoError(t, err)
				expectedLocation := createValidLocation(t, tc.expectedX, tc.expectedY)
				assert.Equal(t, expectedLocation, c.Location())
			})
		}
	})

	t.Run("should handle boundary movements", func(t *testing.T) {
		testCases := []struct {
			name             string
			startX, startY   kernel.Coordinate
			targetX, targetY kernel.Coordinate
			speed            int
		}{
			{
				name:   "from corner to corner",
				startX: 1, startY: 1,
				targetX: 10, targetY: 10,
				speed: 10,
			},
			{
				name:   "along top edge",
				startX: 1, startY: 10,
				targetX: 10, targetY: 10,
				speed: 5,
			},
			{
				name:   "along bottom edge",
				startX: 10, startY: 1,
				targetX: 1, targetY: 1,
				speed: 8,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				startLocation := createValidLocation(t, tc.startX, tc.startY)
				c, err := courier.NewCourier(kernel.NewUUID(), "Test", tc.speed, startLocation)
				require.NoError(t, err)

				targetLocation := createValidLocation(t, tc.targetX, tc.targetY)

				err = c.Move(targetLocation)

				require.NoError(t, err)
				// Should not go out of bounds (1-10 range)
				newLocation := c.Location()
				assert.GreaterOrEqual(t, int(newLocation.X()), 1)
				assert.LessOrEqual(t, int(newLocation.X()), 10)
				assert.GreaterOrEqual(t, int(newLocation.Y()), 1)
				assert.LessOrEqual(t, int(newLocation.Y()), 10)
			})
		}
	})

	t.Run("should make multiple moves to reach distant target", func(t *testing.T) {
		startLocation := createValidLocation(t, 1, 1)
		c, err := courier.NewCourier(kernel.NewUUID(), "Test", 2, startLocation)
		require.NoError(t, err)

		targetLocation := createValidLocation(t, 7, 5) // 6 steps X, 4 steps Y = 10 total

		// First move: should move 2 steps toward target
		err = c.Move(targetLocation)
		require.NoError(t, err)
		assert.Equal(t, createValidLocation(t, 3, 1), c.Location())

		// Second move: should move 2 more steps
		err = c.Move(targetLocation)
		require.NoError(t, err)
		assert.Equal(t, createValidLocation(t, 5, 1), c.Location())

		// Third move: should move remaining 2 X steps
		err = c.Move(targetLocation)
		require.NoError(t, err)
		assert.Equal(t, createValidLocation(t, 7, 1), c.Location())

		// Fourth move: should start Y movement
		err = c.Move(targetLocation)
		require.NoError(t, err)
		assert.Equal(t, createValidLocation(t, 7, 3), c.Location())

		// Fifth move: should reach target
		err = c.Move(targetLocation)
		require.NoError(t, err)
		assert.Equal(t, targetLocation, c.Location())
	})
}

func TestCourier_CalculateTimeToLocation(t *testing.T) {
	t.Run("should return 0 for same location", func(t *testing.T) {
		location := createValidLocation(t, 5, 5)
		c, err := courier.NewCourier(kernel.NewUUID(), "Test", 3, location)
		require.NoError(t, err)

		time, err := c.CalculateTimeToLocation(location)

		require.NoError(t, err)
		assert.InDelta(t, 0.0, time, 0.0001)
	})

	t.Run("should return error for invalid target location", func(t *testing.T) {
		c := createValidCourier(t)
		var invalidLocation kernel.Location

		time, err := c.CalculateTimeToLocation(invalidLocation)

		require.Error(t, err)
		assert.InDelta(t, 0.0, time, 0.0001)
		assert.Contains(t, err.Error(), "location must be created")
	})

	t.Run("should calculate correct time for various distances and speeds", func(t *testing.T) {
		testCases := []struct {
			name             string
			startX, startY   kernel.Coordinate
			targetX, targetY kernel.Coordinate
			speed            int
			expectedTime     float64
		}{
			{
				name:   "distance 4, speed 2",
				startX: 1, startY: 1,
				targetX: 3, targetY: 3,
				speed:        2,
				expectedTime: 2.0, // Manhattan distance: |1-3| + |1-3| = 4, time: 4/2 = 2
			},
			{
				name:   "distance 6, speed 3",
				startX: 2, startY: 3,
				targetX: 5, targetY: 6,
				speed:        3,
				expectedTime: 2.0, // Distance: |2-5| + |3-6| = 6, time: 6/3 = 2
			},
			{
				name:   "fractional time",
				startX: 1, startY: 1,
				targetX: 4, targetY: 1,
				speed:        2,
				expectedTime: 1.5, // Distance: 3, time: 3/2 = 1.5
			},
			{
				name:   "high speed, short distance",
				startX: 5, startY: 5,
				targetX: 6, targetY: 6,
				speed:        10,
				expectedTime: 0.2, // Distance: 2, time: 2/10 = 0.2
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				startLocation := createValidLocation(t, tc.startX, tc.startY)
				c, err := courier.NewCourier(kernel.NewUUID(), "Test", tc.speed, startLocation)
				require.NoError(t, err)

				targetLocation := createValidLocation(t, tc.targetX, tc.targetY)

				time, err := c.CalculateTimeToLocation(targetLocation)

				require.NoError(t, err)
				assert.InEpsilon(t, tc.expectedTime, time, 0.0001)
			})
		}
	})

	t.Run("should handle maximum distance on grid", func(t *testing.T) {
		startLocation := createValidLocation(t, 1, 1)
		targetLocation := createValidLocation(t, 10, 10)
		c, err := courier.NewCourier(kernel.NewUUID(), "Test", 3, startLocation)
		require.NoError(t, err)

		time, err := c.CalculateTimeToLocation(targetLocation)

		require.NoError(t, err)
		// Distance: |1-10| + |1-10| = 18, time: 18/3 = 6
		assert.InEpsilon(t, 6.0, time, 0.0001)
	})
}

func TestCourier_CanTakeOrder(t *testing.T) {
	t.Run("should return true for valid order that fits in storage", func(t *testing.T) {
		c := createValidCourier(t)      // Has default storage with 10 volume
		order := createValidOrder(t, 5) // Order with volume 5

		canTake, err := c.CanTakeOrder(order)

		require.NoError(t, err)
		assert.True(t, canTake)
	})

	t.Run("should return true for order with exact storage capacity", func(t *testing.T) {
		c := createValidCourier(t)       // Has default storage with 10 volume
		order := createValidOrder(t, 10) // Order with exact capacity

		canTake, err := c.CanTakeOrder(order)

		require.NoError(t, err)
		assert.True(t, canTake)
	})

	t.Run("should return false for order exceeding storage capacity", func(t *testing.T) {
		c := createValidCourier(t)       // Has default storage with 10 volume
		order := createValidOrder(t, 15) // Order exceeds capacity

		canTake, err := c.CanTakeOrder(order)

		require.NoError(t, err)
		assert.False(t, canTake)
	})

	t.Run("should return error for invalid order", func(t *testing.T) {
		c := createValidCourier(t)
		var invalidOrder *order.Order

		canTake, err := c.CanTakeOrder(invalidOrder)

		require.Error(t, err)
		assert.False(t, canTake)
	})

	t.Run("should return false when storage is already occupied", func(t *testing.T) {
		c := createValidCourier(t)
		firstOrder := createValidOrder(t, 8)
		secondOrder := createValidOrder(t, 2) // Small order that would normally fit

		// Take first order
		err := c.TakeOrder(firstOrder)
		require.NoError(t, err)

		// Try to check if second order can be taken
		canTake, err := c.CanTakeOrder(secondOrder)

		require.NoError(t, err)
		assert.False(t, canTake) // Should be false because storage is occupied
	})

	t.Run("should find available storage when courier has multiple storage places", func(t *testing.T) {
		c := createValidCourier(t)

		// Add additional storage
		err := c.AddStoragePlace("Backpack", 15)
		require.NoError(t, err)

		// Occupy first storage
		firstOrder := createValidOrder(t, 10)
		err = c.TakeOrder(firstOrder)
		require.NoError(t, err)

		// Check if new order can be taken (should use second storage)
		secondOrder := createValidOrder(t, 12)
		canTake, err := c.CanTakeOrder(secondOrder)

		require.NoError(t, err)
		assert.True(t, canTake)
	})
}

func TestCourier_TakeOrder(t *testing.T) {
	t.Run("should take order successfully with valid parameters", func(t *testing.T) {
		c := createValidCourier(t)
		order := createValidOrder(t, 8)

		err := c.TakeOrder(order)

		require.NoError(t, err)

		// Verify order is stored
		storagePlaces := c.StoragePlaces()
		require.Len(t, storagePlaces, 1)
		assert.NotNil(t, storagePlaces[0].OrderID())
		assert.True(t, storagePlaces[0].OrderID().IsEqual(order.ID()))
	})

	t.Run("should take order with exact storage capacity", func(t *testing.T) {
		c := createValidCourier(t)
		order := createValidOrder(t, 10) // Exact capacity

		err := c.TakeOrder(order)

		require.NoError(t, err)

		// Verify order is stored
		storagePlaces := c.StoragePlaces()
		require.Len(t, storagePlaces, 1)
		assert.NotNil(t, storagePlaces[0].OrderID())
		assert.True(t, storagePlaces[0].OrderID().IsEqual(order.ID()))
	})

	t.Run("should return error for invalid order", func(t *testing.T) {
		c := createValidCourier(t)
		var invalidOrder *order.Order

		err := c.TakeOrder(invalidOrder)

		require.Error(t, err)

		// Verify no order is stored
		storagePlaces := c.StoragePlaces()
		require.Len(t, storagePlaces, 1)
		assert.Nil(t, storagePlaces[0].OrderID())
	})

	t.Run("should return error when order exceeds storage capacity", func(t *testing.T) {
		c := createValidCourier(t)
		order := createValidOrder(t, 15) // Exceeds capacity

		err := c.TakeOrder(order)

		require.Error(t, err)
		assert.Equal(t, courier.ErrStoragePlaceNotFound, err)

		// Verify no order is stored
		storagePlaces := c.StoragePlaces()
		require.Len(t, storagePlaces, 1)
		assert.Nil(t, storagePlaces[0].OrderID())
	})

	t.Run("should return error when storage is already occupied", func(t *testing.T) {
		c := createValidCourier(t)
		firstOrder := createValidOrder(t, 5)
		secondOrder := createValidOrder(t, 3)

		// Take first order
		err := c.TakeOrder(firstOrder)
		require.NoError(t, err)

		// Try to take second order
		err = c.TakeOrder(secondOrder)

		require.Error(t, err)
		assert.Equal(t, courier.ErrStoragePlaceNotFound, err)

		// Verify first order is still stored
		storagePlaces := c.StoragePlaces()
		require.Len(t, storagePlaces, 1)
		assert.NotNil(t, storagePlaces[0].OrderID())
		assert.True(t, storagePlaces[0].OrderID().IsEqual(firstOrder.ID()))
	})

	t.Run("should use first available storage place", func(t *testing.T) {
		c := createValidCourier(t)

		// Add additional storage places
		err := c.AddStoragePlace("Backpack", 15)
		require.NoError(t, err)
		err = c.AddStoragePlace("Side Bag", 5)
		require.NoError(t, err)

		order := createValidOrder(t, 8)

		err = c.TakeOrder(order)

		require.NoError(t, err)

		// Should use first storage place (default bag with 10 capacity)
		storagePlaces := c.StoragePlaces()
		require.Len(t, storagePlaces, 3)
		assert.NotNil(t, storagePlaces[0].OrderID())
		assert.True(t, storagePlaces[0].OrderID().IsEqual(order.ID()))
		assert.Nil(t, storagePlaces[1].OrderID())
		assert.Nil(t, storagePlaces[2].OrderID())
	})

	t.Run("should use appropriate storage place when first is too small", func(t *testing.T) {
		c := createValidCourier(t)

		// Occupy default storage with large order
		largeOrder := createValidOrder(t, 10)
		err := c.TakeOrder(largeOrder)
		require.NoError(t, err)

		// Add larger storage
		err = c.AddStoragePlace("Large Backpack", 20)
		require.NoError(t, err)

		// Take another large order
		secondOrder := createValidOrder(t, 15)
		err = c.TakeOrder(secondOrder)

		require.NoError(t, err)

		// Should use the large backpack
		storagePlaces := c.StoragePlaces()
		require.Len(t, storagePlaces, 2)
		assert.NotNil(t, storagePlaces[0].OrderID())
		assert.True(t, storagePlaces[0].OrderID().IsEqual(largeOrder.ID()))
		assert.NotNil(t, storagePlaces[1].OrderID())
		assert.True(t, storagePlaces[1].OrderID().IsEqual(secondOrder.ID()))
	})
}

func TestCourier_CompleteOrder(t *testing.T) {
	t.Run("should complete order successfully", func(t *testing.T) {
		c := createValidCourier(t)
		order := createValidOrder(t, 8)

		// Take order first
		err := c.TakeOrder(order)
		require.NoError(t, err)

		// Complete the order
		err = c.CompleteOrder(order.ID())

		require.NoError(t, err)

		// Verify order is no longer stored
		storagePlaces := c.StoragePlaces()
		require.Len(t, storagePlaces, 1)
		assert.Nil(t, storagePlaces[0].OrderID())
	})

	t.Run("should return error for invalid order ID", func(t *testing.T) {
		c := createValidCourier(t)
		order := createValidOrder(t, 8)

		// Take order first
		err := c.TakeOrder(order)
		require.NoError(t, err)

		// Try to complete with invalid ID
		var invalidID kernel.UUID
		err = c.CompleteOrder(invalidID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), kernel.ErrUUIDIsNotConstructed.Error())

		// Verify order is still stored
		storagePlaces := c.StoragePlaces()
		require.Len(t, storagePlaces, 1)
		assert.NotNil(t, storagePlaces[0].OrderID())
		assert.True(t, storagePlaces[0].OrderID().IsEqual(order.ID()))
	})

	t.Run("should return error when order not found", func(t *testing.T) {
		c := createValidCourier(t)
		order := createValidOrder(t, 8)

		// Take order first
		err := c.TakeOrder(order)
		require.NoError(t, err)

		// Try to complete with different order ID
		differentOrderID := kernel.NewUUID()
		err = c.CompleteOrder(differentOrderID)

		require.Error(t, err)
		assert.Equal(t, courier.ErrStoragePlaceNotFound, err)

		// Verify original order is still stored
		storagePlaces := c.StoragePlaces()
		require.Len(t, storagePlaces, 1)
		assert.NotNil(t, storagePlaces[0].OrderID())
		assert.True(t, storagePlaces[0].OrderID().IsEqual(order.ID()))
	})

	t.Run("should return error when no orders are stored", func(t *testing.T) {
		c := createValidCourier(t)
		orderID := kernel.NewUUID()

		err := c.CompleteOrder(orderID)

		require.Error(t, err)
		assert.Equal(t, courier.ErrStoragePlaceNotFound, err)
	})

	t.Run("should complete correct order when multiple orders are stored", func(t *testing.T) {
		c := createValidCourier(t)

		// Add additional storage
		err := c.AddStoragePlace("Backpack", 15)
		require.NoError(t, err)

		// Take two orders
		firstOrder := createValidOrder(t, 8)
		secondOrder := createValidOrder(t, 12)

		err = c.TakeOrder(firstOrder)
		require.NoError(t, err)
		err = c.TakeOrder(secondOrder)
		require.NoError(t, err)

		// Complete first order
		err = c.CompleteOrder(firstOrder.ID())

		require.NoError(t, err)

		// Verify first storage is empty, second still has order
		storagePlaces := c.StoragePlaces()
		require.Len(t, storagePlaces, 2)
		assert.Nil(t, storagePlaces[0].OrderID())
		assert.NotNil(t, storagePlaces[1].OrderID())
		assert.True(t, storagePlaces[1].OrderID().IsEqual(secondOrder.ID()))
	})

	t.Run("should allow taking new order after completion", func(t *testing.T) {
		c := createValidCourier(t)
		firstOrder := createValidOrder(t, 8)
		secondOrder := createValidOrder(t, 6)

		// Take and complete first order
		err := c.TakeOrder(firstOrder)
		require.NoError(t, err)
		err = c.CompleteOrder(firstOrder.ID())
		require.NoError(t, err)

		// Take second order
		err = c.TakeOrder(secondOrder)

		require.NoError(t, err)

		// Verify second order is stored
		storagePlaces := c.StoragePlaces()
		require.Len(t, storagePlaces, 1)
		assert.NotNil(t, storagePlaces[0].OrderID())
		assert.True(t, storagePlaces[0].OrderID().IsEqual(secondOrder.ID()))
	})
}

func TestCourier_AddStoragePlace(t *testing.T) {
	t.Run("should add storage place successfully with valid parameters", func(t *testing.T) {
		c := createValidCourier(t)
		initialCount := len(c.StoragePlaces())

		err := c.AddStoragePlace("Backpack", 20)

		require.NoError(t, err)

		storagePlaces := c.StoragePlaces()
		assert.Len(t, storagePlaces, initialCount+1)

		// Check the new storage place
		newStorage := storagePlaces[len(storagePlaces)-1]
		assert.Equal(t, "Backpack", newStorage.Name())
		assert.Equal(t, 20, newStorage.TotalVolume())
		assert.Nil(t, newStorage.OrderID())
	})

	t.Run("should return error for empty name", func(t *testing.T) {
		c := createValidCourier(t)
		initialCount := len(c.StoragePlaces())

		err := c.AddStoragePlace("", 15)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "name")

		// Verify no storage place was added
		storagePlaces := c.StoragePlaces()
		assert.Len(t, storagePlaces, initialCount)
	})

	t.Run("should return error for invalid volume", func(t *testing.T) {
		c := createValidCourier(t)
		initialCount := len(c.StoragePlaces())

		testCases := []struct {
			name   string
			volume int
		}{
			{"zero volume", 0},
			{"negative volume", -5},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := c.AddStoragePlace("Test Storage", tc.volume)

				require.Error(t, err)
				assert.Contains(t, err.Error(), "totalVolume is invalid")

				// Verify no storage place was added
				storagePlaces := c.StoragePlaces()
				assert.Len(t, storagePlaces, initialCount)
			})
		}
	})

	t.Run("should add multiple storage places", func(t *testing.T) {
		c := createValidCourier(t)

		storageConfigs := []struct {
			name   string
			volume int
		}{
			{"Small Bag", 5},
			{"Medium Backpack", 15},
			{"Large Container", 25},
		}

		for i, config := range storageConfigs {
			err := c.AddStoragePlace(config.name, config.volume)
			require.NoError(t, err)

			storagePlaces := c.StoragePlaces()
			assert.Len(t, storagePlaces, 1+i+1) // +1 for default bag, +i+1 for added ones

			newStorage := storagePlaces[len(storagePlaces)-1]
			assert.Equal(t, config.name, newStorage.Name())
			assert.Equal(t, config.volume, newStorage.TotalVolume())
		}
	})

	t.Run("should handle boundary values for volume", func(t *testing.T) {
		c := createValidCourier(t)

		testCases := []struct {
			name        string
			volume      int
			shouldError bool
		}{
			{"minimum valid volume", 1, false},
			{"small volume", 5, false},
			{"large volume", 1000, false},
			{"very large volume", 999999, false},
			{"zero volume", 0, true},
			{"negative volume", -1, true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				initialCount := len(c.StoragePlaces())
				err := c.AddStoragePlace("Test Storage", tc.volume)

				if tc.shouldError {
					require.Error(t, err)
					storagePlaces := c.StoragePlaces()
					assert.Len(t, storagePlaces, initialCount)
				} else {
					require.NoError(t, err)
					storagePlaces := c.StoragePlaces()
					assert.Len(t, storagePlaces, initialCount+1)

					newStorage := storagePlaces[len(storagePlaces)-1]
					assert.Equal(t, tc.volume, newStorage.TotalVolume())
				}
			})
		}
	})

	t.Run("should create storage places with unique IDs", func(t *testing.T) {
		c := createValidCourier(t)

		err := c.AddStoragePlace("Storage 1", 10)
		require.NoError(t, err)
		err = c.AddStoragePlace("Storage 2", 15)
		require.NoError(t, err)

		storagePlaces := c.StoragePlaces()
		require.Len(t, storagePlaces, 3) // Default + 2 added

		// All IDs should be unique
		for i := range storagePlaces {
			for j := range storagePlaces {
				if j > i {
					assert.False(t, storagePlaces[i].ID().IsEqual(storagePlaces[j].ID()),
						"Storage places should have unique IDs")
				}
			}
		}
	})

	t.Run("should allow adding storage places with same name but different volumes", func(t *testing.T) {
		c := createValidCourier(t)

		err := c.AddStoragePlace("Bag", 10)
		require.NoError(t, err)
		err = c.AddStoragePlace("Bag", 20) // Same name, different volume
		require.NoError(t, err)

		storagePlaces := c.StoragePlaces()
		require.Len(t, storagePlaces, 3) // Default + 2 added

		// Both should exist with same name but different volumes
		addedStorage1 := storagePlaces[1]
		addedStorage2 := storagePlaces[2]
		assert.Equal(t, "Bag", addedStorage1.Name())
		assert.Equal(t, "Bag", addedStorage2.Name())
		assert.Equal(t, 10, addedStorage1.TotalVolume())
		assert.Equal(t, 20, addedStorage2.TotalVolume())
		assert.False(t, addedStorage1.ID().IsEqual(addedStorage2.ID()))
	})
}

func TestCourier_IntegrationScenarios(t *testing.T) {
	t.Run("complete delivery workflow", func(t *testing.T) {
		// Create courier at pickup location
		pickupLocation := createValidLocation(t, 1, 1)
		c, err := courier.NewCourier(kernel.NewUUID(), "Alice", 3, pickupLocation)
		require.NoError(t, err)

		// Create order for delivery
		deliveryLocation := createValidLocation(t, 8, 6)
		order := createValidOrder(t, 8)

		// 1. Check if courier can take order
		canTake, err := c.CanTakeOrder(order)
		require.NoError(t, err)
		assert.True(t, canTake)

		// 2. Take the order
		err = c.TakeOrder(order)
		require.NoError(t, err)

		// 3. Calculate time to delivery location
		deliveryTime, err := c.CalculateTimeToLocation(deliveryLocation)
		require.NoError(t, err)
		expectedDistance := 7 + 5 // |1-8| + |1-6| = 12
		expectedTime := float64(expectedDistance) / 3.0
		assert.InEpsilon(t, expectedTime, deliveryTime, 0.0001)

		// 4. Move toward delivery location (multiple moves required)
		for c.Location() != deliveryLocation {
			oldLocation := c.Location()
			err = c.Move(deliveryLocation)
			require.NoError(t, err)

			// Should make progress toward target
			newDistance, newErr := c.Location().Distance(deliveryLocation)
			require.NoError(t, newErr)
			oldDistance, oldErr := oldLocation.Distance(deliveryLocation)
			require.NoError(t, oldErr)
			assert.LessOrEqual(t, newDistance, oldDistance)
		}

		// 5. Complete delivery
		err = c.CompleteOrder(order.ID())
		require.NoError(t, err)

		// 6. Verify courier is at delivery location and order is completed
		assert.Equal(t, deliveryLocation, c.Location())
		storagePlaces := c.StoragePlaces()
		assert.Nil(t, storagePlaces[0].OrderID())
	})

	t.Run("multiple orders with different storage places", func(t *testing.T) {
		c := createValidCourier(t)

		// Add varied storage places
		err := c.AddStoragePlace("Small Bag", 5)
		require.NoError(t, err)
		err = c.AddStoragePlace("Large Backpack", 20)
		require.NoError(t, err)

		// Create orders that will fit in different storage places
		smallOrder := createValidOrder(t, 4)  // Will fit in Small Bag (5 capacity)
		mediumOrder := createValidOrder(t, 8) // Will fit in Default Bag (10 capacity)
		largeOrder := createValidOrder(t, 15) // Will fit in Large Backpack (20 capacity)

		// Take orders - first-fit algorithm should place them in first available storage
		err = c.TakeOrder(smallOrder) // Should go to Default Bag (10 capacity) - first available
		require.NoError(t, err)
		err = c.TakeOrder(mediumOrder) // Should go to Small Bag (5 capacity), but won't fit, so Large Backpack
		require.NoError(t, err)
		err = c.TakeOrder(largeOrder) // Default and Large are occupied, Small Bag can't fit it
		require.Error(t, err)         // This should fail

		// Verify only two orders are stored
		storagePlaces := c.StoragePlaces()
		require.Len(t, storagePlaces, 3)
		assert.NotNil(t, storagePlaces[0].OrderID()) // Default bag has smallOrder
		assert.Nil(t, storagePlaces[1].OrderID())    // Small bag is empty (can't fit mediumOrder)
		assert.NotNil(t, storagePlaces[2].OrderID()) // Large backpack has mediumOrder

		// Complete one order to make space
		err = c.CompleteOrder(smallOrder.ID()) // Complete from default bag
		require.NoError(t, err)

		// Now we can take the large order
		// Should go to Default Bag (10 capacity) but won't fit, then Large Backpack but occupied, then fail
		err = c.TakeOrder(largeOrder)
		require.Error(t, err) // Still can't fit because Large Backpack is occupied

		// Complete the medium order to free large storage
		err = c.CompleteOrder(mediumOrder.ID())
		require.NoError(t, err)

		// Now large order should fit
		err = c.TakeOrder(largeOrder)
		require.NoError(t, err)

		// Verify final state
		storagePlaces = c.StoragePlaces()
		assert.Nil(t, storagePlaces[0].OrderID())    // Default bag empty
		assert.Nil(t, storagePlaces[1].OrderID())    // Small bag empty
		assert.NotNil(t, storagePlaces[2].OrderID()) // Large backpack has largeOrder
		assert.True(t, storagePlaces[2].OrderID().IsEqual(largeOrder.ID()))
	})

	t.Run("movement with order delivery simulation", func(t *testing.T) {
		startLocation := createValidLocation(t, 2, 2)
		c, err := courier.NewCourier(kernel.NewUUID(), "Bob", 2, startLocation)
		require.NoError(t, err)

		// Multiple delivery locations
		deliveryPoints := []kernel.Location{
			createValidLocation(t, 5, 2), // 3 steps east
			createValidLocation(t, 5, 6), // 4 steps north
			createValidLocation(t, 1, 6), // 4 steps west
			createValidLocation(t, 1, 1), // 5 steps south
		}

		for i, deliveryPoint := range deliveryPoints {
			// Create and take order
			order := createValidOrder(t, 5)
			err = c.TakeOrder(order)
			require.NoError(t, err)

			// Move to delivery point (may require multiple moves)
			maxMoves := 20 // Safety limit
			moves := 0
			for c.Location() != deliveryPoint && moves < maxMoves {
				err = c.Move(deliveryPoint)
				require.NoError(t, err)
				moves++
			}

			// Should reach destination
			assert.Equal(t, deliveryPoint, c.Location(), "Should reach delivery point %d", i+1)

			// Complete delivery
			err = c.CompleteOrder(order.ID())
			require.NoError(t, err)

			// Verify storage is empty
			storagePlaces := c.StoragePlaces()
			assert.Nil(t, storagePlaces[0].OrderID())
		}
	})
}

func TestCourier_EdgeCases(t *testing.T) {
	t.Run("should handle maximum storage capacity", func(t *testing.T) {
		c := createValidCourier(t)

		// Fill storage to maximum
		order := createValidOrder(t, 10) // Exact capacity
		err := c.TakeOrder(order)
		require.NoError(t, err)

		// Try to add another order (should fail)
		anotherOrder := createValidOrder(t, 1)
		canTake, err := c.CanTakeOrder(anotherOrder)
		require.NoError(t, err)
		assert.False(t, canTake)

		err = c.TakeOrder(anotherOrder)
		require.Error(t, err)
	})

	t.Run("should handle courier with high speed", func(t *testing.T) {
		startLocation := createValidLocation(t, 1, 1)
		highSpeedCourier, err := courier.NewCourier(kernel.NewUUID(), "Flash", 18, startLocation)
		require.NoError(t, err)

		targetLocation := createValidLocation(t, 10, 10) // Max distance: 18 steps

		err = highSpeedCourier.Move(targetLocation)
		require.NoError(t, err)

		// Should reach target in one move
		assert.Equal(t, targetLocation, highSpeedCourier.Location())
	})

	t.Run("should handle courier with speed 1", func(t *testing.T) {
		startLocation := createValidLocation(t, 1, 1)
		slowCourier, err := courier.NewCourier(kernel.NewUUID(), "Turtle", 1, startLocation)
		require.NoError(t, err)

		targetLocation := createValidLocation(t, 3, 1) // 2 steps away

		// First move: should move 1 step
		err = slowCourier.Move(targetLocation)
		require.NoError(t, err)
		assert.Equal(t, createValidLocation(t, 2, 1), slowCourier.Location())

		// Second move: should reach target
		err = slowCourier.Move(targetLocation)
		require.NoError(t, err)
		assert.Equal(t, targetLocation, slowCourier.Location())
	})

	t.Run("should handle grid boundary movements", func(t *testing.T) {
		testCases := []struct {
			name             string
			startX, startY   kernel.Coordinate
			targetX, targetY kernel.Coordinate
			speed            int
		}{
			{"top-left to bottom-right", 1, 1, 10, 10, 20},
			{"bottom-right to top-left", 10, 10, 1, 1, 20},
			{"along top edge", 1, 10, 10, 10, 15},
			{"along bottom edge", 10, 1, 1, 1, 15},
			{"vertical movement only", 5, 1, 5, 10, 12},
			{"horizontal movement only", 1, 5, 10, 5, 12},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				startLocation := createValidLocation(t, tc.startX, tc.startY)
				c, err := courier.NewCourier(kernel.NewUUID(), "Boundary", tc.speed, startLocation)
				require.NoError(t, err)

				targetLocation := createValidLocation(t, tc.targetX, tc.targetY)

				err = c.Move(targetLocation)
				require.NoError(t, err)

				// Should stay within bounds
				newLocation := c.Location()
				assert.GreaterOrEqual(t, int(newLocation.X()), 1)
				assert.LessOrEqual(t, int(newLocation.X()), 10)
				assert.GreaterOrEqual(t, int(newLocation.Y()), 1)
				assert.LessOrEqual(t, int(newLocation.Y()), 10)
			})
		}
	})

	t.Run("should handle empty and full storage transitions", func(t *testing.T) {
		c := createValidCourier(t)

		// Initially empty
		canTake, err := c.CanTakeOrder(createValidOrder(t, 5))
		require.NoError(t, err)
		assert.True(t, canTake)

		// Fill storage
		order := createValidOrder(t, 10)
		err = c.TakeOrder(order)
		require.NoError(t, err)

		// Now full
		canTake, err = c.CanTakeOrder(createValidOrder(t, 1))
		require.NoError(t, err)
		assert.False(t, canTake)

		// Empty again
		err = c.CompleteOrder(order.ID())
		require.NoError(t, err)

		// Should be available again
		canTake, err = c.CanTakeOrder(createValidOrder(t, 8))
		require.NoError(t, err)
		assert.True(t, canTake)
	})

	t.Run("should handle orders with volume 1", func(t *testing.T) {
		c := createValidCourier(t)
		minOrder := createValidOrder(t, 1) // Minimum volume

		canTake, err := c.CanTakeOrder(minOrder)
		require.NoError(t, err)
		assert.True(t, canTake)

		err = c.TakeOrder(minOrder)
		require.NoError(t, err)

		err = c.CompleteOrder(minOrder.ID())
		require.NoError(t, err)

		// Should work without issues
		storagePlaces := c.StoragePlaces()
		assert.Nil(t, storagePlaces[0].OrderID())
	})

	t.Run("should handle rapid order cycling", func(t *testing.T) {
		c := createValidCourier(t)

		// Rapidly take and complete many orders
		for range 10 {
			order := createValidOrder(t, 5)

			err := c.TakeOrder(order)
			require.NoError(t, err)

			err = c.CompleteOrder(order.ID())
			require.NoError(t, err)

			// Storage should be empty after each cycle
			storagePlaces := c.StoragePlaces()
			assert.Nil(t, storagePlaces[0].OrderID())
		}
	})
}

func TestCourier_ErrorResilience(t *testing.T) {
	t.Run("operations should not change state on error", func(t *testing.T) {
		c := createValidCourier(t)
		originalLocation := c.Location()
		order := createValidOrder(t, 8)

		// Take valid order
		err := c.TakeOrder(order)
		require.NoError(t, err)

		// Try invalid operations that should not change state
		var invalidLocation kernel.Location
		var invalidOrderID kernel.UUID

		// Invalid move operation
		err = c.Move(invalidLocation)
		require.Error(t, err)
		assert.Equal(t, originalLocation, c.Location()) // Location unchanged

		// Invalid order operations
		err = c.TakeOrder(createValidOrder(t, 5)) // Should fail - storage occupied
		require.Error(t, err)

		err = c.CompleteOrder(invalidOrderID) // Should fail - invalid ID
		require.Error(t, err)

		// Original order should still be stored
		storagePlaces := c.StoragePlaces()
		assert.NotNil(t, storagePlaces[0].OrderID())
		assert.True(t, storagePlaces[0].OrderID().IsEqual(order.ID()))
	})

	t.Run("should maintain invariants after failed operations", func(t *testing.T) {
		c := createValidCourier(t)

		// Try to add invalid storage place
		err := c.AddStoragePlace("", -5) // Invalid name and volume
		require.Error(t, err)

		// Should still have only default storage
		storagePlaces := c.StoragePlaces()
		assert.Len(t, storagePlaces, 1)
		assert.Equal(t, "Сумка", storagePlaces[0].Name())

		// Try to take invalid order
		var invalidOrder *order.Order
		err = c.TakeOrder(invalidOrder)
		require.Error(t, err)

		// Storage should remain empty
		assert.Nil(t, storagePlaces[0].OrderID())

		// Courier should still be valid
		require.NoError(t, c.Validate())
	})
}
