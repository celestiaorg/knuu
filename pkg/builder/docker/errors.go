package docker

import (
	"fmt"
)

type Error struct {
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err == e {
		return e.Message
	}

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
	ErrFailedToListBuildxBuilders = &Error{Code: "FailedToListBuildxBuilders", Message: "failed to list buildx builders"}
	ErrRunCommandFailed           = &Error{Code: "RunCommandFailed", Message: "failed to run command"}
	ErrFailedToCreateBuilder      = &Error{Code: "FailedToCreateBuilder", Message: "failed to create buildx builder"}
	ErrFailedToBuildImage         = &Error{Code: "FailedToBuildImage", Message: "failed to build image"}
	ErrFailedToPushImage          = &Error{Code: "FailedToPushImage", Message: "failed to push image"}
	ErrFailedToRemoveContextDir   = &Error{Code: "FailedToRemoveContextDir", Message: "failed to remove context directory"}
	ErrGitContextNotSupported     = &Error{Code: "GitContextNotSupported", Message: "git context is not supported in the docker builder"}
)
