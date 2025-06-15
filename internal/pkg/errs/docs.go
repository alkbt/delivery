// Package errs provides standardized error types for the delivery application.
// It implements a consistent pattern for error creation, formatting, and unwrapping
// that is used throughout the application.
//
// The package includes several error types for common error scenarios:
//   - ValueIsRequiredError: For when a required value is missing
//   - ValueIsInvalidError: For when a value is invalid
//   - ObjectNotFoundError: For when an object cannot be found
//   - Other specialized error types for specific validation failures
//
// Each error type follows a consistent pattern:
//   - A sentinel error variable (e.g., ErrValueIsRequired)
//   - A struct type with fields for error details
//   - Constructor functions with and without cause
//   - Error() method for formatting the error message
//   - Unwrap() method for error wrapping/unwrapping support
//
// This standardized approach to error handling improves error reporting,
// makes error handling more consistent, and enables better error classification
// and handling throughout the application.
package errs
