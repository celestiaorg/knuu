package errors

import (
	"errors"
	"fmt"
)

type Error struct {
	code    string
	message string
	err     error
	params  []interface{}
}

func New(code, message string) *Error {
	return &Error{
		code:    code,
		message: message,
	}
}

// It returns true if the error is the same as the given error
// it checks the error codes for comparison
func Is(err1, err2 error) bool {
	if err1 == nil || err2 == nil {
		return false
	}
	e1, ok1 := err1.(*Error)
	e2, ok2 := err2.(*Error)
	return ok1 && ok2 && e1.code == e2.code
}

func (e *Error) Error() string {
	// We need to keep this condition to avoid infinite recursion
	if errors.Is(e.err, e) {
		return e.message
	}

	msg := fmt.Sprintf(e.message, e.params...)
	if e.err != nil {
		return fmt.Sprintf("%s: %v", msg, e.err)
	}
	return msg
}

func (e *Error) Wrap(err error) *Error {
	e.err = errors.Join(e.err, err)
	return e
}

func (e *Error) WithParams(params ...interface{}) *Error {
	e.params = params
	return e
}

func (e *Error) Code() string {
	return e.code
}

func (e *Error) Message() string {
	return e.message
}
