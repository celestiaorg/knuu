package docker

import (
	"github.com/celestiaorg/knuu/pkg/errors"
)

type Error = errors.Error

var (
	ErrFailedToListBuildxBuilders = errors.New("FailedToListBuildxBuilders", "failed to list buildx builders")
	ErrRunCommandFailed           = errors.New("RunCommandFailed", "failed to run command")
	ErrFailedToCreateBuilder      = errors.New("FailedToCreateBuilder", "failed to create buildx builder")
	ErrFailedToBuildImage         = errors.New("FailedToBuildImage", "failed to build image")
	ErrFailedToPushImage          = errors.New("FailedToPushImage", "failed to push image")
	ErrFailedToRemoveContextDir   = errors.New("FailedToRemoveContextDir", "failed to remove context directory")
	ErrGitContextNotSupported     = errors.New("GitContextNotSupported", "git context is not supported in the docker builder")
)
