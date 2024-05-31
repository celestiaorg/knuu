package errors

import (
	"errors"
	"testing"
	"time"

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

func TestError_RecursiveError(t *testing.T) {
	done := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Test panicked: %v", r)
				done <- false
			}
		}()

		err := New("123", "recursive error")
		err.err = err // Simulate recursion by setting err to itself

		expected := "recursive error"
		assert.Equal(t, expected, err.Error())
		done <- true
	}()

	select {
	case <-done:
		// Test completed
	case <-time.After(1 * time.Second):
		t.Error("Test timed out, possible infinite recursion")
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

func TestIs(t *testing.T) {
	tests := []struct {
		name     string
		err1     error
		err2     error
		expected bool
	}{
		{
			name:     "both nil errors",
			err1:     nil,
			err2:     nil,
			expected: true,
		},
		{
			name:     "first error nil",
			err1:     nil,
			err2:     New("123", "error 123"),
			expected: false,
		},
		{
			name:     "second error nil",
			err1:     New("123", "error 123"),
			err2:     nil,
			expected: false,
		},
		{
			name:     "different types of errors",
			err1:     errors.New("standard error"),
			err2:     New("123", "error 123"),
			expected: false,
		},
		{
			name:     "same custom error codes",
			err1:     New("123", "error 123"),
			err2:     New("123", "another error 123"),
			expected: true,
		},
		{
			name:     "different custom error codes",
			err1:     New("123", "error 123"),
			err2:     New("456", "error 456"),
			expected: false,
		},
		{
			name:     "one standard error, one custom error",
			err1:     errors.New("standard error"),
			err2:     New("123", "error 123"),
			expected: false,
		},
		{
			name:     "nil comparison with non-nil error",
			err1:     nil,
			err2:     errors.New("standard error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, errors.Is(tt.err1, tt.err2))
		})
	}
}
