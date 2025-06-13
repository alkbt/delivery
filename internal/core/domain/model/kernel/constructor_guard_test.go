package kernel_test

import (
	"errors"
	"testing"

	"delivery/internal/core/domain/model/kernel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConstructorGuard(t *testing.T) {
	t.Run("creates_properly_constructed_guard", func(t *testing.T) {
		// When
		guard := kernel.NewConstructorGuard()

		// Then
		assert.NotNil(t, guard)
		
		// Test with custom error
		customError := errors.New("test object not constructed")
		assert.NoError(t, guard.Validate(customError))
		
		// Test with nil error (should use default)
		assert.NoError(t, guard.Validate(nil))
	})
}

func TestConstructorGuard_Validate(t *testing.T) {
	t.Run("properly_constructed_guard_returns_nil", func(t *testing.T) {
		// Given
		guard := kernel.NewConstructorGuard()
		customError := errors.New("not constructed")

		// When
		err := guard.Validate(customError)

		// Then
		assert.NoError(t, err)
	})

	t.Run("zero_value_guard_returns_custom_error", func(t *testing.T) {
		// Given
		var guard kernel.ConstructorGuard // zero value
		expectedError := errors.New("entity not constructed")

		// When
		err := guard.Validate(expectedError)

		// Then
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
	})

	t.Run("zero_value_guard_returns_default_error_when_nil", func(t *testing.T) {
		// Given
		var guard kernel.ConstructorGuard // zero value

		// When
		err := guard.Validate(nil)

		// Then
		assert.Error(t, err)
		assert.Equal(t, kernel.ErrDefaultConstructorGuard, err)
	})
}

// TestConstructorGuardUsageExample demonstrates how ConstructorGuard should be used
// in a domain object to enforce constructor usage
func TestConstructorGuardUsageExample(t *testing.T) {
	// Define a sample domain object that uses ConstructorGuard
	type Money struct {
		amount   int
		currency string
		guard    kernel.ConstructorGuard
	}

	var ErrMoneyNotConstructed = errors.New("Money must be created via NewMoney")

	NewMoney := func(amount int, currency string) (Money, error) {
		if amount < 0 {
			return Money{}, errors.New("amount cannot be negative")
		}
		if currency == "" {
			return Money{}, errors.New("currency is required")
		}
		return Money{
			amount:   amount,
			currency: currency,
			guard:    kernel.NewConstructorGuard(),
		}, nil
	}

	ValidateMoney := func(m Money) error {
		return m.guard.Validate(ErrMoneyNotConstructed)
	}

	t.Run("valid_construction_through_constructor", func(t *testing.T) {
		// When
		money, err := NewMoney(100, "USD")

		// Then
		require.NoError(t, err)
		assert.NoError(t, ValidateMoney(money))
		assert.Equal(t, 100, money.amount)
		assert.Equal(t, "USD", money.currency)
	})

	t.Run("zero_value_construction_validation", func(t *testing.T) {
		// Given
		var money Money // zero value

		// When
		err := ValidateMoney(money)

		// Then
		// Zero value Money has zero value guard which returns the error we pass
		assert.Error(t, err)
		assert.Equal(t, ErrMoneyNotConstructed, err)
	})

	t.Run("constructor_validates_business_rules", func(t *testing.T) {
		// Test negative amount
		_, err := NewMoney(-100, "USD")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "amount cannot be negative")

		// Test empty currency
		_, err = NewMoney(100, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "currency is required")
	})
}

// TestConstructorGuardRealWorldExample shows a better pattern using embedded types
func TestConstructorGuardRealWorldExample(t *testing.T) {
	// Define error once
	var ErrProductNotConstructed = errors.New("Product must be created via NewProduct")

	// Define a guard-aware base type
	type guardedProduct struct {
		guard kernel.ConstructorGuard
	}

	newGuardedProduct := func() guardedProduct {
		return guardedProduct{
			guard: kernel.NewConstructorGuard(),
		}
	}

	validateGuardedProduct := func(g guardedProduct) error {
		return g.guard.Validate(ErrProductNotConstructed)
	}

	// Define the actual domain object
	type Product struct {
		guardedProduct
		id    string
		name  string
		price int
	}

	NewProduct := func(id, name string, price int) (Product, error) {
		if id == "" {
			return Product{}, errors.New("product ID is required")
		}
		if name == "" {
			return Product{}, errors.New("product name is required")
		}
		if price < 0 {
			return Product{}, errors.New("product price cannot be negative")
		}
		return Product{
			guardedProduct: newGuardedProduct(),
			id:             id,
			name:           name,
			price:          price,
		}, nil
	}

	t.Run("valid_product_construction", func(t *testing.T) {
		// When
		product, err := NewProduct("123", "Laptop", 999)

		// Then
		require.NoError(t, err)
		assert.NoError(t, validateGuardedProduct(product.guardedProduct))
		assert.Equal(t, "123", product.id)
		assert.Equal(t, "Laptop", product.name)
		assert.Equal(t, 999, product.price)
	})

	t.Run("zero_value_product_fails_validation", func(t *testing.T) {
		// Given
		var product Product // zero value

		// When
		err := validateGuardedProduct(product.guardedProduct)

		// Then
		// Zero value has zero value guard which returns the error we pass
		assert.Error(t, err)
		assert.Equal(t, ErrProductNotConstructed, err)
	})
}

// TestConstructorGuardWithMultipleErrors demonstrates using ConstructorGuard
// with different error types and messages
func TestConstructorGuardWithMultipleErrors(t *testing.T) {
	testCases := []struct {
		name          string
		expectedError error
	}{
		{
			name:          "order_not_constructed_error",
			expectedError: errors.New("Order must be created via NewOrder"),
		},
		{
			name:          "product_not_constructed_error",
			expectedError: errors.New("Product must be created via NewProduct factory method"),
		},
		{
			name:          "customer_not_constructed_error",
			expectedError: errors.New("Customer requires proper initialization through constructor"),
		},
		{
			name:          "nil_error_uses_default",
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Given
			guard := kernel.NewConstructorGuard()

			// When
			err := guard.Validate(tc.expectedError)

			// Then
			assert.NoError(t, err, "Properly constructed guard should not return error")
		})
	}
}

// TestConstructorGuardDefaultError verifies the default error behavior
func TestConstructorGuardDefaultError(t *testing.T) {
	t.Run("nil_error_uses_default_for_zero_value", func(t *testing.T) {
		// Given
		var guard kernel.ConstructorGuard // zero value

		// When
		err := guard.Validate(nil)

		// Then
		assert.Error(t, err)
		assert.Equal(t, kernel.ErrDefaultConstructorGuard, err)
	})

	t.Run("default_error_constant_has_meaningful_message", func(t *testing.T) {
		// Then
		assert.NotNil(t, kernel.ErrDefaultConstructorGuard)
		assert.Contains(t, kernel.ErrDefaultConstructorGuard.Error(), "constructor")
		assert.Equal(t, "object must be created via its constructor", kernel.ErrDefaultConstructorGuard.Error())
	})
}

// BenchmarkConstructorGuard measures the performance overhead of using ConstructorGuard
func BenchmarkConstructorGuard(b *testing.B) {
	b.Run("NewConstructorGuard", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = kernel.NewConstructorGuard()
		}
	})

	b.Run("Validate_Success", func(b *testing.B) {
		guard := kernel.NewConstructorGuard()
		err := errors.New("not constructed")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = guard.Validate(err)
		}
	})

	b.Run("Validate_ZeroValue", func(b *testing.B) {
		var guard kernel.ConstructorGuard
		err := errors.New("not constructed")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = guard.Validate(err)
		}
	})
}

// TestConstructorGuardConcurrency verifies that ConstructorGuard is safe for concurrent use
func TestConstructorGuardConcurrency(t *testing.T) {
	guard := kernel.NewConstructorGuard()
	validationError := errors.New("not constructed")
	
	// Run multiple goroutines that validate the guard concurrently
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			for j := 0; j < 1000; j++ {
				err := guard.Validate(validationError)
				assert.NoError(t, err)
			}
			done <- true
		}()
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 100; i++ {
		<-done
	}
}

// TestConstructorGuardImmutability verifies that ConstructorGuard is immutable
func TestConstructorGuardImmutability(t *testing.T) {
	t.Run("guard_fields_are_not_modifiable", func(t *testing.T) {
		// Given
		originalError := errors.New("original error")
		guard := kernel.NewConstructorGuard()

		// When
		// Try to create another guard
		anotherError := errors.New("another error")
		_ = kernel.NewConstructorGuard()

		// Then
		// Original guard should still validate successfully
		assert.NoError(t, guard.Validate(originalError))
		assert.NoError(t, guard.Validate(anotherError))
	})

	t.Run("guard_can_be_safely_passed_by_value", func(t *testing.T) {
		// Given
		guard := kernel.NewConstructorGuard()
		testError := errors.New("test error")

		// When
		guardCopy := guard // Pass by value

		// Then
		assert.NoError(t, guard.Validate(testError))
		assert.NoError(t, guardCopy.Validate(testError))
	})
}