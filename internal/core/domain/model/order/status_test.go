package order_test

import (
	"fmt"
	"testing"

	"delivery/internal/core/domain/model/order"
	"delivery/internal/pkg/errs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatus_Constants(t *testing.T) {
	t.Run("should have correct enum values", func(t *testing.T) {
		assert.Equal(t, 0, int(order.Unknown))
		assert.Equal(t, 1, int(order.Created))
		assert.Equal(t, 2, int(order.Assigned))
		assert.Equal(t, 3, int(order.Completed))
	})

	t.Run("should have distinct values", func(t *testing.T) {
		statuses := []order.Status{
			order.Unknown,
			order.Created,
			order.Assigned,
			order.Completed,
		}

		for i, status1 := range statuses {
			for j, status2 := range statuses {
				if i != j {
					assert.NotEqual(t, status1, status2,
						"statuses at indices %d and %d should be different", i, j)
				}
			}
		}
	})
}

func TestStatus_Validate(t *testing.T) {
	t.Run("should validate valid statuses", func(t *testing.T) {
		validStatuses := []order.Status{
			order.Created,
			order.Assigned,
			order.Completed,
		}

		for _, status := range validStatuses {
			t.Run(fmt.Sprintf("should validate %s status", status.String()), func(t *testing.T) {
				err := status.Validate()
				require.NoError(t, err)
			})
		}
	})

	t.Run("should reject Unknown status", func(t *testing.T) {
		err := order.Unknown.Validate()

		require.Error(t, err)
		assert.IsType(t, &errs.ValueIsInvalidError{}, err)
		assert.Contains(t, err.Error(), "status is invalid")
		assert.Contains(t, err.Error(), "0 is not a valid status")
	})

	t.Run("should reject invalid status values", func(t *testing.T) {
		invalidStatuses := []order.Status{
			order.Status(-1),
			order.Status(4),
			order.Status(100),
			order.Status(-999),
		}

		for _, status := range invalidStatuses {
			t.Run(fmt.Sprintf("should reject status value %d", int(status)), func(t *testing.T) {
				err := status.Validate()

				require.Error(t, err)
				assert.IsType(t, &errs.ValueIsInvalidError{}, err)
				assert.Contains(t, err.Error(), "status is invalid")
				assert.Contains(t, err.Error(), fmt.Sprintf("%d is not a valid status", int(status)))
			})
		}
	})
}

func TestStatus_String(t *testing.T) {
	t.Run("should return correct string for valid statuses", func(t *testing.T) {
		testCases := []struct {
			status   order.Status
			expected string
		}{
			{order.Created, "Created"},
			{order.Assigned, "Assigned"},
			{order.Completed, "Completed"},
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("should return %s for %d", tc.expected, int(tc.status)), func(t *testing.T) {
				result := tc.status.String()
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("should return Unknown for invalid statuses", func(t *testing.T) {
		invalidStatuses := []order.Status{
			order.Unknown,
			order.Status(-1),
			order.Status(4),
			order.Status(100),
		}

		for _, status := range invalidStatuses {
			t.Run(fmt.Sprintf("should return Unknown for status value %d", int(status)), func(t *testing.T) {
				result := status.String()
				assert.Equal(t, "Unknown", result)
			})
		}
	})

	t.Run("should implement fmt.Stringer interface", func(t *testing.T) {
		status := order.Created
		formatted := status.String()
		assert.Equal(t, "Created", formatted)
	})
}

func TestStatus_Assign(t *testing.T) {
	t.Run("should allow transition from Created to Assigned", func(t *testing.T) {
		status := order.Created

		newStatus, err := status.Assign()

		require.NoError(t, err)
		assert.Equal(t, order.Assigned, newStatus)
	})

	t.Run("should allow transition from Assigned to Assigned (reassignment)", func(t *testing.T) {
		status := order.Assigned

		newStatus, err := status.Assign()

		require.NoError(t, err)
		assert.Equal(t, order.Assigned, newStatus)
	})

	t.Run("should reject transition from Completed to Assigned", func(t *testing.T) {
		status := order.Completed

		newStatus, err := status.Assign()

		require.Error(t, err)
		assert.Equal(t, order.Status(0), newStatus)
		assert.IsType(t, &errs.ValueIsInvalidError{}, err)
		assert.Contains(t, err.Error(), "status is invalid")
		assert.Contains(t, err.Error(), "Completed is not a valid status to assign")
	})

	t.Run("should reject transition from Unknown to Assigned", func(t *testing.T) {
		status := order.Unknown

		newStatus, err := status.Assign()

		require.Error(t, err)
		assert.Equal(t, order.Status(0), newStatus)
		assert.IsType(t, &errs.ValueIsInvalidError{}, err)
		assert.Contains(t, err.Error(), "status is invalid")
		assert.Contains(t, err.Error(), "Unknown is not a valid status to assign")
	})

	t.Run("should reject transition from invalid status values", func(t *testing.T) {
		invalidStatuses := []order.Status{
			order.Status(-1),
			order.Status(4),
			order.Status(100),
		}

		for _, status := range invalidStatuses {
			t.Run(fmt.Sprintf("should reject transition from status %d", int(status)), func(t *testing.T) {
				newStatus, err := status.Assign()

				require.Error(t, err)
				assert.Equal(t, order.Status(0), newStatus)
				assert.Contains(t, err.Error(), "is not a valid status to assign")
			})
		}
	})
}

func TestStatus_Complete(t *testing.T) {
	t.Run("should allow transition from Assigned to Completed", func(t *testing.T) {
		status := order.Assigned

		newStatus, err := status.Complete()

		require.NoError(t, err)
		assert.Equal(t, order.Completed, newStatus)
	})

	t.Run("should reject transition from Created to Completed", func(t *testing.T) {
		status := order.Created

		newStatus, err := status.Complete()

		require.Error(t, err)
		assert.Equal(t, order.Status(0), newStatus)
		assert.IsType(t, &errs.ValueIsInvalidError{}, err)
		assert.Contains(t, err.Error(), "status is invalid")
		assert.Contains(t, err.Error(), "Created is not a valid status to complete")
	})

	t.Run("should reject transition from Completed to Completed", func(t *testing.T) {
		status := order.Completed

		newStatus, err := status.Complete()

		require.Error(t, err)
		assert.Equal(t, order.Status(0), newStatus)
		assert.IsType(t, &errs.ValueIsInvalidError{}, err)
		assert.Contains(t, err.Error(), "status is invalid")
		assert.Contains(t, err.Error(), "Completed is not a valid status to complete")
	})

	t.Run("should reject transition from Unknown to Completed", func(t *testing.T) {
		status := order.Unknown

		newStatus, err := status.Complete()

		require.Error(t, err)
		assert.Equal(t, order.Status(0), newStatus)
		assert.IsType(t, &errs.ValueIsInvalidError{}, err)
		assert.Contains(t, err.Error(), "status is invalid")
		assert.Contains(t, err.Error(), "Unknown is not a valid status to complete")
	})

	t.Run("should reject transition from invalid status values", func(t *testing.T) {
		invalidStatuses := []order.Status{
			order.Status(-1),
			order.Status(4),
			order.Status(100),
		}

		for _, status := range invalidStatuses {
			t.Run(fmt.Sprintf("should reject transition from status %d", int(status)), func(t *testing.T) {
				newStatus, err := status.Complete()

				require.Error(t, err)
				assert.Equal(t, order.Status(0), newStatus)
				assert.Contains(t, err.Error(), "is not a valid status to complete")
			})
		}
	})
}

func TestStatus_StateMachine(t *testing.T) {
	t.Run("should follow valid state transitions", func(t *testing.T) {
		// Test full valid workflow: Created -> Assigned -> Completed
		status := order.Created

		// Created -> Assigned
		status, err := status.Assign()
		require.NoError(t, err)
		assert.Equal(t, order.Assigned, status)

		// Assigned -> Completed
		status, err = status.Complete()
		require.NoError(t, err)
		assert.Equal(t, order.Completed, status)
	})

	t.Run("should handle reassignment workflow", func(t *testing.T) {
		// Test reassignment: Created -> Assigned -> Assigned
		status := order.Created

		// Created -> Assigned
		status, err := status.Assign()
		require.NoError(t, err)
		assert.Equal(t, order.Assigned, status)

		// Assigned -> Assigned (reassignment)
		status, err = status.Assign()
		require.NoError(t, err)
		assert.Equal(t, order.Assigned, status)

		// Still can complete
		status, err = status.Complete()
		require.NoError(t, err)
		assert.Equal(t, order.Completed, status)
	})

	t.Run("should prevent invalid transition sequences", func(t *testing.T) {
		// Test Created -> Completed (should fail)
		status := order.Created
		_, err := status.Complete()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Created is not a valid status to complete")

		// Test Completed -> Assigned (should fail)
		status = order.Completed
		_, err = status.Assign()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Completed is not a valid status to assign")

		// Test Completed -> Completed (should fail)
		_, err = status.Complete()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Completed is not a valid status to complete")
	})
}

func TestStatus_Immutability(t *testing.T) {
	t.Run("should not modify original status during transitions", func(t *testing.T) {
		originalStatus := order.Created

		// Attempt transition
		newStatus, err := originalStatus.Assign()
		require.NoError(t, err)

		// Original should be unchanged
		assert.Equal(t, order.Created, originalStatus)
		assert.Equal(t, order.Assigned, newStatus)
		assert.NotEqual(t, originalStatus, newStatus)
	})

	t.Run("should not modify original status on failed transitions", func(t *testing.T) {
		originalStatus := order.Completed

		// Attempt invalid transition
		_, err := originalStatus.Assign()
		require.Error(t, err)

		// Original should be unchanged
		assert.Equal(t, order.Completed, originalStatus)
	})
}

func TestStatus_EdgeCases(t *testing.T) {
	t.Run("should handle zero value status", func(t *testing.T) {
		var status order.Status // Zero value is Unknown

		assert.Equal(t, order.Unknown, status)
		assert.Equal(t, "Unknown", status.String())
		require.Error(t, status.Validate())
	})

	t.Run("should handle type conversion edge cases", func(t *testing.T) {
		// Test conversion from int
		status := order.Status(1)
		assert.Equal(t, order.Created, status)
		assert.Equal(t, "Created", status.String())
		require.NoError(t, status.Validate())

		// Test conversion from invalid int
		invalidStatus := order.Status(999)
		assert.Equal(t, "Unknown", invalidStatus.String())
		require.Error(t, invalidStatus.Validate())
	})

	t.Run("should handle boundary values", func(t *testing.T) {
		// Test just below valid range
		belowRange := order.Status(-1)
		assert.Equal(t, "Unknown", belowRange.String())
		require.Error(t, belowRange.Validate())

		// Test just above valid range
		aboveRange := order.Status(4)
		assert.Equal(t, "Unknown", aboveRange.String())
		require.Error(t, aboveRange.Validate())
	})
}

func TestStatus_Documentation(t *testing.T) {
	t.Run("should demonstrate state machine transitions correctly", func(t *testing.T) {
		// This test validates the state machine diagram in the documentation
		// Created ──┬──> Assigned ──> Completed
		//           │        │
		//           └────────┘
		//      (reassignment allowed)

		// Path 1: Created -> Assigned -> Completed
		status := order.Created
		status, err := status.Assign()
		require.NoError(t, err)
		status, err = status.Complete()
		require.NoError(t, err)
		assert.Equal(t, order.Completed, status)

		// Path 2: Created -> Assigned -> Assigned -> Completed
		status = order.Created
		status, err = status.Assign()
		require.NoError(t, err)
		status, err = status.Assign() // Reassignment
		require.NoError(t, err)
		status, err = status.Complete()
		require.NoError(t, err)
		assert.Equal(t, order.Completed, status)

		// Invalid path: Created -> Completed (should fail)
		status = order.Created
		_, err = status.Complete()
		require.Error(t, err)

		// Invalid path: Completed -> Assigned (should fail)
		status = order.Completed
		_, err = status.Assign()
		require.Error(t, err)
	})
}

func TestStatus_ValidateAssign(t *testing.T) {
	t.Run("should allow assignment from valid statuses", func(t *testing.T) {
		validStatuses := []order.Status{
			order.Created,
			order.Assigned,
		}

		for _, status := range validStatuses {
			t.Run(fmt.Sprintf("should allow assignment from %s status", status.String()), func(t *testing.T) {
				err := status.ValidateAssign()
				require.NoError(t, err)
			})
		}
	})

	t.Run("should reject assignment from invalid statuses", func(t *testing.T) {
		invalidStatuses := []order.Status{
			order.Completed,
			order.Unknown,
		}

		for _, status := range invalidStatuses {
			t.Run(fmt.Sprintf("should reject assignment from %s status", status.String()), func(t *testing.T) {
				err := status.ValidateAssign()

				require.Error(t, err)
				assert.IsType(t, &errs.ValueIsInvalidError{}, err)
				assert.Contains(t, err.Error(), "status is invalid")
				assert.Contains(t, err.Error(), fmt.Sprintf("%s is not a valid status to assign", status.String()))
			})
		}
	})

	t.Run("should reject assignment from arbitrary invalid status values", func(t *testing.T) {
		invalidStatuses := []order.Status{
			order.Status(-1),
			order.Status(4),
			order.Status(100),
			order.Status(-999),
		}

		for _, status := range invalidStatuses {
			t.Run(fmt.Sprintf("should reject assignment from status value %d", int(status)), func(t *testing.T) {
				err := status.ValidateAssign()

				require.Error(t, err)
				assert.IsType(t, &errs.ValueIsInvalidError{}, err)
				assert.Contains(t, err.Error(), "status is invalid")
				assert.Contains(t, err.Error(), "is not a valid status to assign")
			})
		}
	})

	t.Run("should have consistent behavior with Assign method", func(t *testing.T) {
		allStatuses := []order.Status{
			order.Unknown,
			order.Created,
			order.Assigned,
			order.Completed,
			order.Status(-1),
			order.Status(4),
		}

		for _, status := range allStatuses {
			t.Run(fmt.Sprintf("consistency check for status %s (%d)", status.String(), int(status)),
				func(t *testing.T) {
					validateErr := status.ValidateAssign()
					_, assignErr := status.Assign()

					// Both methods should agree on assignability
					if validateErr == nil {
						assert.NoError(t, assignErr, "ValidateAssign passed but Assign failed")
					} else {
						assert.Error(t, assignErr, "ValidateAssign failed but Assign succeeded")
					}
				})
		}
	})
}

func TestStatus_Consistency(t *testing.T) {
	t.Run("should have consistent String() and Validate() behavior", func(t *testing.T) {
		allPossibleStatuses := []order.Status{
			order.Status(-100),
			order.Status(-1),
			order.Unknown,
			order.Created,
			order.Assigned,
			order.Completed,
			order.Status(4),
			order.Status(100),
		}

		for _, status := range allPossibleStatuses {
			t.Run(fmt.Sprintf("status %d", int(status)), func(t *testing.T) {
				str := status.String()
				err := status.Validate()

				if str == "Unknown" {
					require.Error(t, err, "status with String() 'Unknown' should fail validation")
				} else {
					require.NoError(t, err, "status with valid String() should pass validation")
				}
			})
		}
	})

	t.Run("should have bidirectional consistency between transitions", func(t *testing.T) {
		// If status X can transition to Y via Assign(), then Y should be order.Assigned
		testStatuses := []order.Status{order.Created, order.Assigned}
		for _, status := range testStatuses {
			newStatus, err := status.Assign()
			if err == nil {
				assert.Equal(t, order.Assigned, newStatus)
			}
		}

		// If status X can transition to Y via Complete(), then Y should be order.Completed
		status := order.Assigned
		newStatus, err := status.Complete()
		require.NoError(t, err)
		assert.Equal(t, order.Completed, newStatus)
	})
}
