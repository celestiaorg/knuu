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
			name:     "simple error with message only",
			errorObj: New("123", "simple message"),
			expected: "simple message",
		},
		{
			name:     "error with parameters",
			errorObj: New("123", "message with %s and %d").WithParams("string", 42),
			expected: "message with string and 42",
		},
		{
			name: "error with nested error",
			errorObj: &Error{
				code:    "123",
				message: "parent message",
				err:     errors.New("child error"),
			},
			expected: "parent message: child error",
		},
		{
			name: "error with nested error and parameters",
			errorObj: &Error{
				code:    "123",
				message: "message with %s",
				params:  []interface{}{"parameter"},
				err:     errors.New("child error"),
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
	tests := []struct {
		name      string
		errorObj  *Error
		wrapError error
		expected  string
	}{
		{
			name: "wrap no error",
			errorObj: &Error{
				message: "test message",
				err:     errors.New("initial error"),
			},
			wrapError: nil,
			expected:  "test message: initial error",
		},
		{
			name: "wrap an error",
			errorObj: &Error{
				message: "test message",
				err:     errors.New("initial error"),
			},
			wrapError: errors.New("wrapped error"),
			expected:  "test message: initial error\nwrapped error",
		},
		{
			name: "wrap a joined error",
			errorObj: &Error{
				message: "test message",
				err:     errors.New("initial error"),
			},
			wrapError: errors.Join(errors.New("wrapped error 1"), errors.New("wrapped error 2")),
			expected:  "test message: initial error\nwrapped error 1\nwrapped error 2",
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
				code:    "123",
				message: "message with %s and %d",
			},
			params:   []interface{}{"string", 42},
			expected: "message with string and 42",
		},
		{
			name: "overwrite existing parameters",
			errorObj: &Error{
				code:    "123",
				message: "message with %s and %d",
				params:  []interface{}{"old", 0},
			},
			params:   []interface{}{"new", 100},
			expected: "message with new and 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errWithParams := tt.errorObj.WithParams(tt.params...)
			require.NotNil(t, errWithParams)
			assert.Equal(t, tt.params, errWithParams.params)
			assert.Equal(t, tt.expected, errWithParams.Error())
		})
	}
}
