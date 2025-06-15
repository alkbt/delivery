package errs_test

import (
	"errors"
	"testing"

	"delivery/internal/pkg/errs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObjectNotFoundError(t *testing.T) {
	t.Run("NewObjectNotFoundError", func(t *testing.T) {
		err := errs.NewObjectNotFoundError("userId", "123")

		assert.Equal(t, "userId", err.ParamName)
		assert.Equal(t, "123", err.ID)
		require.NoError(t, err.Cause)
		assert.Equal(t, "object not found: 123", err.Error())
		assert.Equal(t, errs.ErrObjectNotFound, err.Unwrap())
	})

	t.Run("NewObjectNotFoundErrorWithCause", func(t *testing.T) {
		cause := errors.New("database connection failed")
		err := errs.NewObjectNotFoundErrorWithCause("userId", "123", cause)

		assert.Equal(t, "userId", err.ParamName)
		assert.Equal(t, "123", err.ID)
		assert.Equal(t, cause, err.Cause)
		assert.Equal(t,
			"object not found: param is: userId, ID is: 123 (cause: database connection failed)",
			err.Error())
		assert.Equal(t, errs.ErrObjectNotFound, err.Unwrap())
	})

	t.Run("Error with different ID types", func(t *testing.T) {
		err := errs.NewObjectNotFoundError("orderId", 456)
		assert.Equal(t, "object not found: %!s(int=456)", err.Error())
	})
}

func TestValueIsInvalidError(t *testing.T) {
	t.Run("NewValueIsInvalidError", func(t *testing.T) {
		err := errs.NewValueIsInvalidError("email")

		assert.Equal(t, "email", err.ParamName)
		require.NoError(t, err.Cause)
		assert.Equal(t, "value is invalid: email", err.Error())
		assert.Equal(t, errs.ErrValueIsInvalid, err.Unwrap())
	})

	t.Run("NewValueIsInvalidErrorWithCause", func(t *testing.T) {
		cause := errors.New("invalid format")
		err := errs.NewValueIsInvalidErrorWithCause("email", cause)

		assert.Equal(t, "email", err.ParamName)
		assert.Equal(t, cause, err.Cause)
		assert.Equal(t, "value is invalid: email (cause: invalid format)", err.Error())
		assert.Equal(t, errs.ErrValueIsInvalid, err.Unwrap())
	})
}

func TestValueIsOutOfRangeError(t *testing.T) {
	t.Run("NewValueIsOutOfRangeError", func(t *testing.T) {
		err := errs.NewValueIsOutOfRangeError("age", 150, 0, 120)

		assert.Equal(t, "age", err.ParamName)
		assert.Equal(t, 150, err.Value)
		assert.Equal(t, 0, err.Min)
		assert.Equal(t, 120, err.Max)
		require.NoError(t, err.Cause)
		assert.Equal(t, "value is invalid: 150 is age, min value is 0, max value is 120", err.Error())
		assert.Equal(t, errs.ErrValueIsOutOfRange, err.Unwrap())
	})

	t.Run("NewValueIsOutOfRangeErrorWithCause", func(t *testing.T) {
		cause := errors.New("validation failed")
		err := errs.NewValueIsOutOfRangeErrorWithCause("score", -5, 0, 100, cause)

		assert.Equal(t, "score", err.ParamName)
		assert.Equal(t, -5, err.Value)
		assert.Equal(t, 0, err.Min)
		assert.Equal(t, 100, err.Max)
		assert.Equal(t, cause, err.Cause)
		assert.Equal(t,
			"value is invalid: -5 is score, min value is 0, max value is 100 (cause: validation failed)",
			err.Error())
		assert.Equal(t, errs.ErrValueIsOutOfRange, err.Unwrap())
	})

	t.Run("sanitize function with newlines", func(t *testing.T) {
		err := errs.NewValueIsOutOfRangeError("text", "hello\nworld", 0, 10)
		assert.Contains(t, err.Error(), "hello world")
		assert.NotContains(t, err.Error(), "\n")
	})
}

func TestValueIsRequiredError(t *testing.T) {
	t.Run("NewValueIsRequiredError", func(t *testing.T) {
		err := errs.NewValueIsRequiredError("username")

		assert.Equal(t, "username", err.ParamName)
		require.NoError(t, err.Cause)
		assert.Equal(t, "value is required: username", err.Error())
		assert.Equal(t, errs.ErrValueIsRequired, err.Unwrap())
	})

	t.Run("NewValueIsRequiredErrorWithCause", func(t *testing.T) {
		cause := errors.New("missing required field")
		err := errs.NewValueIsRequiredErrorWithCause("username", cause)

		assert.Equal(t, "username", err.ParamName)
		assert.Equal(t, cause, err.Cause)
		assert.Equal(t, "value is required: username (cause: missing required field)", err.Error())
		assert.Equal(t, errs.ErrValueIsRequired, err.Unwrap())
	})
}

func TestVersionIsInvalidError(t *testing.T) {
	t.Run("NewVersionIsInvalidError", func(t *testing.T) {
		cause := errors.New("invalid semver")
		err := errs.NewVersionIsInvalidError("version", cause)

		assert.Equal(t, "version", err.ParamName)
		assert.Equal(t, cause, err.Cause)
		assert.Equal(t, "version is invalid: version (cause: invalid semver)", err.Error())
		assert.Equal(t, errs.ErrVersionIsInvalid, err.Unwrap())
	})

	t.Run("NewVersionIsInvalidErrorWithCause", func(t *testing.T) {
		err := errs.NewVersionIsInvalidErrorWithCause("version")

		assert.Equal(t, "version", err.ParamName)
		require.NoError(t, err.Cause)
		assert.Equal(t, "version is invalid: version", err.Error())
		assert.Equal(t, errs.ErrVersionIsInvalid, err.Unwrap())
	})
}

func TestSentinelErrors(t *testing.T) {
	t.Run("sentinel errors are defined", func(t *testing.T) {
		require.Error(t, errs.ErrObjectNotFound)
		require.Error(t, errs.ErrValueIsInvalid)
		require.Error(t, errs.ErrValueIsOutOfRange)
		require.Error(t, errs.ErrValueIsRequired)
		require.Error(t, errs.ErrVersionIsInvalid)
	})

	t.Run("error messages match expectations", func(t *testing.T) {
		assert.Equal(t, "object not found", errs.ErrObjectNotFound.Error())
		assert.Equal(t, "value is invalid", errs.ErrValueIsInvalid.Error())
		assert.Equal(t, "value is out of range", errs.ErrValueIsOutOfRange.Error())
		assert.Equal(t, "value is required", errs.ErrValueIsRequired.Error())
		assert.Equal(t, "version is invalid", errs.ErrVersionIsInvalid.Error())
	})
}

func TestErrorsCanBeUnwrapped(t *testing.T) {
	t.Run("errors.Is works with custom errors", func(t *testing.T) {
		objectNotFoundErr := errs.NewObjectNotFoundError("userId", "123")
		require.ErrorIs(t, objectNotFoundErr, errs.ErrObjectNotFound)

		valueInvalidErr := errs.NewValueIsInvalidError("email")
		require.ErrorIs(t, valueInvalidErr, errs.ErrValueIsInvalid)

		valueOutOfRangeErr := errs.NewValueIsOutOfRangeError("age", 150, 0, 120)
		require.ErrorIs(t, valueOutOfRangeErr, errs.ErrValueIsOutOfRange)

		valueRequiredErr := errs.NewValueIsRequiredError("username")
		require.ErrorIs(t, valueRequiredErr, errs.ErrValueIsRequired)

		versionInvalidErr := errs.NewVersionIsInvalidError("version", errors.New("test"))
		require.ErrorIs(t, versionInvalidErr, errs.ErrVersionIsInvalid)
	})
}
