package errs

import (
	"errors"
	"fmt"
	"strings"
)

var ErrValueIsOutOfRange = errors.New("value is out of range")

type ValueIsOutOfRangeError struct {
	ParamName string
	Value     any
	Min       any
	Max       any
	Cause     error
}

func NewValueIsOutOfRangeErrorWithCause(
	paramName string, value any, minVal any, maxVal any, cause error,
) *ValueIsOutOfRangeError {
	return &ValueIsOutOfRangeError{
		ParamName: paramName,
		Value:     value,
		Min:       minVal,
		Max:       maxVal,
		Cause:     cause,
	}
}

func NewValueIsOutOfRangeError(paramName string, value any, minVal any, maxVal any) *ValueIsOutOfRangeError {
	return &ValueIsOutOfRangeError{
		ParamName: paramName,
		Value:     value,
		Min:       minVal,
		Max:       maxVal,
	}
}

func (e *ValueIsOutOfRangeError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s is %v, min value is %v, max value is %v (cause: %v)",
			ErrValueIsInvalid, sanitize(e.Value), e.ParamName, e.Min, e.Max, e.Cause)
	}
	return fmt.Sprintf("%s: %s is %v, min value is %v, max value is %v",
		ErrValueIsInvalid, sanitize(e.Value), e.ParamName, e.Min, e.Max)
}

func (e *ValueIsOutOfRangeError) Unwrap() error {
	return ErrValueIsOutOfRange
}

func sanitize(input interface{}) string {
	str := fmt.Sprintf("%v", input)
	return strings.ReplaceAll(str, "\n", " ")
}
