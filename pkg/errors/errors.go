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

func (e *Error) Error() string {
	if e.err == e {
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
