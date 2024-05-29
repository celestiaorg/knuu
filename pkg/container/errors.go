package container

import (
	"github.com/celestiaorg/knuu/pkg/errors"
)

type Error = errors.Error

var (
	ErrCreatingDockerClient           = &Error{Code: "CreatingDockerClient", Message: "failed to create docker client"}
	ErrFailedToCreateContextDir       = &Error{Code: "FailedToCreateContextDir", Message: "failed to create context directory"}
	ErrNoImageNameProvided            = &Error{Code: "NoImageNameProvided", Message: "no image name provided, push before reading"}
	ErrFailedToCreateContainer        = &Error{Code: "FailedToCreateContainer", Message: "failed to create container"}
	ErrFailedToStopContainer          = &Error{Code: "FailedToStopContainer", Message: "failed to stop container"}
	ErrFailedToRemoveContainer        = &Error{Code: "FailedToRemoveContainer", Message: "failed to remove container"}
	ErrFailedToStartContainer         = &Error{Code: "FailedToStartContainer", Message: "failed to start container"}
	ErrFailedToCopyFileFromContainer  = &Error{Code: "FailedToCopyFileFromContainer", Message: "failed to copy file from container"}
	ErrFailedToReadFromTar            = &Error{Code: "FailedToReadFromTar", Message: "failed to read from tar"}
	ErrFailedToReadFileFromTar        = &Error{Code: "FailedToReadFileFromTar", Message: "failed to read file from tar"}
	ErrFileNotFoundInTar              = &Error{Code: "FileNotFoundInTar", Message: "file not found in tar"}
	ErrFailedToWriteDockerfile        = &Error{Code: "FailedToWriteDockerfile", Message: "failed to write Dockerfile"}
	ErrFailedToGetBuildContext        = &Error{Code: "FailedToGetBuildContext", Message: "failed to get build context"}
	ErrFailedToGetDefaultCacheOptions = &Error{Code: "FailedToGetDefaultCacheOptions", Message: "failed to get default cache options"}
	ErrHashingDockerfile              = &Error{Code: "HashingDockerfile", Message: "error hashing Dockerfile content"}
	ErrReadingFile                    = &Error{Code: "ReadingFile", Message: "error reading file: %s"}
	ErrHashingFile                    = &Error{Code: "HashingFile", Message: "error hashing file %s"}
	ErrHashingBuildContext            = &Error{Code: "HashingBuildContext", Message: "error hashing build context"}
)
