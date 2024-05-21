package preloader

import (
	"fmt"
)

type Error struct {
	Code    string
	Message string
	Err     error
	Params  []interface{}
}

func (e *Error) Error() string {
	if e.Err == e {
		return e.Message
	}

	msg := fmt.Sprintf(e.Message, e.Params...)
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", msg, e.Err)
	}
	return msg
}

func (e *Error) Wrap(err error) error {
	e.Err = err
	return e
}

func (e *Error) WithParams(params ...interface{}) *Error {
	e.Params = params
	return e
}

var (
	ErrGeneratingK8sNameForPreloader = &Error{Code: "GeneratingK8sNameForPreloader", Message: "error generating k8s name for preloader"}
)
