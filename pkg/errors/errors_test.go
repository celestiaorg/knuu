package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		errorObj *Error
		expected string
	}{
		{
			name: "simple error with message only",
			errorObj: &Error{
				Code:    "123",
				Message: "simple message",
			},
			expected: "simple message",
		},
		{
			name: "error with parameters",
			errorObj: &Error{
				Code:    "123",
				Message: "message with %s and %d",
				Params:  []interface{}{"string", 42},
			},
			expected: "message with string and 42",
		},
		{
			name: "error with nested error",
			errorObj: &Error{
				Code:    "123",
				Message: "parent message",
				Err:     errors.New("child error"),
			},
			expected: "parent message: child error",
		},
		{
			name: "error with nested error and parameters",
			errorObj: &Error{
				Code:    "123",
				Message: "message with %s",
				Params:  []interface{}{"parameter"},
				Err:     errors.New("child error"),
			},
			expected: "message with parameter: child error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.errorObj.Error())
		})
	}
}

func TestError_Wrap(t *testing.T) {
	var (
		initialError = errors.New("initial error")
		wrappedError = errors.New("wrapped error")
	)

	tests := []struct {
		name      string
		errorObj  *Error
		wrapError error
		expected  string
	}{
		{
			name:      "wrap no error",
			errorObj:  &Error{Message: "test message", Err: initialError},
			wrapError: nil,
			expected:  "test message: initial error",
		},
		{
			name:      "wrap an error",
			errorObj:  &Error{Message: "test message", Err: initialError},
			wrapError: wrappedError,
			expected:  "test message: initial error\nwrapped error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorObj.Wrap(tt.wrapError)
			assert.Equal(t, tt.expected, err.Error())
		})
	}
}

func TestError_WithParams(t *testing.T) {
	tests := []struct {
		name     string
		errorObj *Error
		params   []interface{}
		expected string
	}{
		{
			name: "add parameters to message",
			errorObj: &Error{
				Code:    "123",
				Message: "message with %s and %d",
			},
			params:   []interface{}{"string", 42},
			expected: "message with string and 42",
		},
		{
			name: "overwrite existing parameters",
			errorObj: &Error{
				Code:    "123",
				Message: "message with %s and %d",
				Params:  []interface{}{"old", 0},
			},
			params:   []interface{}{"new", 100},
			expected: "message with new and 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errWithParams := tt.errorObj.WithParams(tt.params...)
			require.NotNil(t, errWithParams)
			assert.Equal(t, tt.params, errWithParams.Params)
			assert.Equal(t, tt.expected, errWithParams.Error())
		})
	}
}
