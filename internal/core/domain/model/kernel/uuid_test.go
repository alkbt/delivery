package kernel_test

import (
	"testing"

	"delivery/internal/core/domain/model/kernel"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUUID(t *testing.T) {
	t.Run("should create a new UUID", func(t *testing.T) {
		id := kernel.NewUUID()

		assert.NotEmpty(t, id.String())
		assert.NoError(t, id.Validate())
		assert.NotEqual(t, "00000000-0000-0000-0000-000000000000", id.String())
	})

	t.Run("should create unique UUIDs", func(t *testing.T) {
		id1 := kernel.NewUUID()
		id2 := kernel.NewUUID()

		assert.NotEqual(t, id1.String(), id2.String())
		assert.False(t, id1.IsEqual(id2))
	})
}

func TestUUIDFromString(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"

	t.Run("should create UUID from valid string", func(t *testing.T) {
		id, err := kernel.UUIDFromString(validUUID)

		require.NoError(t, err)
		assert.Equal(t, validUUID, id.String())
		assert.NoError(t, id.Validate())
	})

	t.Run("should accept UUID with braces", func(t *testing.T) {
		bracedUUID := "{550e8400-e29b-41d4-a716-446655440000}"
		id, err := kernel.UUIDFromString(bracedUUID)

		require.NoError(t, err)
		assert.Equal(t, validUUID, id.String())
	})

	t.Run("should accept UUID with urn prefix", func(t *testing.T) {
		urnUUID := "urn:uuid:550e8400-e29b-41d4-a716-446655440000"
		id, err := kernel.UUIDFromString(urnUUID)

		require.NoError(t, err)
		assert.Equal(t, validUUID, id.String())
	})

	t.Run("should accept UUID without hyphens", func(t *testing.T) {
		nohyphenUUID := "550e8400e29b41d4a716446655440000"
		id, err := kernel.UUIDFromString(nohyphenUUID)

		require.NoError(t, err)
		assert.Equal(t, validUUID, id.String())
	})

	t.Run("should return error for invalid UUID format", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"", "invalid UUID format"},
			{"not-a-uuid", "invalid UUID format"},
			{"550e8400-e29b-41d4-a716", "invalid UUID format"},
			{"550e8400-e29b-41d4-a716-446655440000-extra", "invalid UUID format"},
			{"zzze8400-e29b-41d4-a716-446655440000", "invalid UUID format"},
			{"550e8400-e29b-41d4-a716-44665544000g", "invalid UUID format"},
		}

		for _, tc := range testCases {
			_, err := kernel.UUIDFromString(tc.input)
			assert.Error(t, err, "expected error for input: %s", tc.input)
			assert.Contains(t, err.Error(), tc.expected)
		}
	})
}

func TestUUIDFromBytes(t *testing.T) {
	validBytes := []byte{
		0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00,
	}

	t.Run("should create UUID from valid bytes", func(t *testing.T) {
		id, err := kernel.UUIDFromBytes(validBytes)

		require.NoError(t, err)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", id.String())
		assert.NoError(t, id.Validate())
	})

	t.Run("should return error for invalid byte length", func(t *testing.T) {
		invalidBytes := []byte{0x55, 0x0e, 0x84}
		_, err := kernel.UUIDFromBytes(invalidBytes)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid UUID format")
	})

	t.Run("should return error for nil bytes", func(t *testing.T) {
		nilBytes := []byte{
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		}
		_, err := kernel.UUIDFromBytes(nilBytes)

		assert.Error(t, err)
		assert.Equal(t, kernel.ErrUUIDIsNotConstructed, err)
	})
}

func TestUUID_String(t *testing.T) {
	t.Run("should return string representation", func(t *testing.T) {
		id := kernel.NewUUID()
		str := id.String()

		assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, str)
	})

	t.Run("should return consistent string representation", func(t *testing.T) {
		id, _ := kernel.UUIDFromString("550e8400-e29b-41d4-a716-446655440000")

		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", id.String())
		assert.Equal(t, id.String(), id.String())
	})
}

func TestUUID_Bytes(t *testing.T) {
	t.Run("should return underlying uuid.UUID", func(t *testing.T) {
		id := kernel.NewUUID()
		bytes := id.Bytes()

		assert.IsType(t, uuid.UUID{}, bytes)
		assert.Equal(t, id.String(), bytes.String())
	})
}

func TestUUID_IsEqual(t *testing.T) {
	t.Run("should return true for equal UUIDs", func(t *testing.T) {
		id1, _ := kernel.UUIDFromString("550e8400-e29b-41d4-a716-446655440000")
		id2, _ := kernel.UUIDFromString("550e8400-e29b-41d4-a716-446655440000")

		assert.True(t, id1.IsEqual(id2))
		assert.True(t, id2.IsEqual(id1))
	})

	t.Run("should return false for different UUIDs", func(t *testing.T) {
		id1 := kernel.NewUUID()
		id2 := kernel.NewUUID()

		assert.False(t, id1.IsEqual(id2))
		assert.False(t, id2.IsEqual(id1))
	})

	t.Run("should handle zero value comparison", func(t *testing.T) {
		var id1 kernel.UUID
		var id2 kernel.UUID
		id3 := kernel.NewUUID()

		assert.True(t, id1.IsEqual(id2))
		assert.False(t, id1.IsEqual(id3))
	})
}

func TestUUID_Validate(t *testing.T) {
	t.Run("should return nil for valid UUID", func(t *testing.T) {
		id := kernel.NewUUID()
		assert.NoError(t, id.Validate())
	})

	t.Run("should return error for zero value UUID", func(t *testing.T) {
		var id kernel.UUID
		err := id.Validate()

		assert.Error(t, err)
		assert.Equal(t, kernel.ErrUUIDIsNotConstructed, err)
	})

	t.Run("should return error for nil UUID", func(t *testing.T) {
		id, _ := kernel.UUIDFromString("00000000-0000-0000-0000-000000000000")
		err := id.Validate()

		assert.Error(t, err)
		assert.Equal(t, kernel.ErrUUIDIsNotConstructed, err)
	})
}

func TestUUID_UsageInStruct(t *testing.T) {
	type Order struct {
		ID kernel.UUID
	}

	t.Run("should work as struct field", func(t *testing.T) {
		order := Order{
			ID: kernel.NewUUID(),
		}

		assert.NoError(t, order.ID.Validate())
		assert.NotEmpty(t, order.ID.String())
	})

	t.Run("should detect uninitialized field", func(t *testing.T) {
		var order Order
		assert.Error(t, order.ID.Validate())
	})
}

func TestUUID_Immutability(t *testing.T) {
	t.Run("modifying Bytes() result does not affect original UUID", func(t *testing.T) {
		// Create a UUID and store its string representation
		original := kernel.NewUUID()
		originalString := original.String()

		// Get the bytes and modify them
		bytes := original.Bytes()
		for i := range bytes {
			bytes[i] = 0xFF
		}

		// Verify the original UUID is unchanged
		assert.Equal(t, originalString, original.String())
		assert.NoError(t, original.Validate())

		// The modified bytes should create a different UUID
		modifiedUUID := uuid.UUID(bytes)
		assert.NotEqual(t, original.String(), modifiedUUID.String())
	})
}
