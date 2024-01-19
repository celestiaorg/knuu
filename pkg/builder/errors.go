package builder

import (
	"fmt"
)

type Error struct {
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Wrap(err error) error {
	e.Err = err
	return e
}

var (
	ErrBuildContextEmpty = &Error{Code: "BuildContextEmpty", Message: "build context cannot be empty"}
)
