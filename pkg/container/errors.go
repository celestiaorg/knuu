package container

import (
	"github.com/celestiaorg/knuu/pkg/errors"
)

type Error = errors.Error

var (
	ErrCreatingDockerClient           = errors.New("CreatingDockerClient", "failed to create docker client")
	ErrFailedToCreateContextDir       = errors.New("FailedToCreateContextDir", "failed to create context directory")
	ErrNoImageNameProvided            = errors.New("NoImageNameProvided", "no image name provided, push before reading")
	ErrFailedToCreateContainer        = errors.New("FailedToCreateContainer", "failed to create container")
	ErrFailedToStopContainer          = errors.New("FailedToStopContainer", "failed to stop container")
	ErrFailedToRemoveContainer        = errors.New("FailedToRemoveContainer", "failed to remove container")
	ErrFailedToStartContainer         = errors.New("FailedToStartContainer", "failed to start container")
	ErrFailedToCopyFileFromContainer  = errors.New("FailedToCopyFileFromContainer", "failed to copy file from container")
	ErrFailedToReadFromTar            = errors.New("FailedToReadFromTar", "failed to read from tar")
	ErrFailedToReadFileFromTar        = errors.New("FailedToReadFileFromTar", "failed to read file from tar")
	ErrFileNotFoundInTar              = errors.New("FileNotFoundInTar", "file not found in tar")
	ErrFailedToWriteDockerfile        = errors.New("FailedToWriteDockerfile", "failed to write Dockerfile")
	ErrFailedToGetBuildContext        = errors.New("FailedToGetBuildContext", "failed to get build context")
	ErrFailedToGetDefaultCacheOptions = errors.New("FailedToGetDefaultCacheOptions", "failed to get default cache options")
	ErrHashingDockerfile              = errors.New("HashingDockerfile", "error hashing Dockerfile content")
	ErrReadingFile                    = errors.New("ReadingFile", "error reading file: %s")
	ErrHashingFile                    = errors.New("HashingFile", "error hashing file %s")
	ErrHashingBuildContext            = errors.New("HashingBuildContext", "error hashing build context")
	ErrImageNameEmpty                 = errors.New("ImageNameEmpty", "image name is empty")
	ErrBuildContextEmpty              = errors.New("BuildContextEmpty", "build context is empty")
	ErrImageBuilderNotSet             = errors.New("ImageBuilderNotSet", "image builder is not set")
	ErrLoggerEmpty                    = errors.New("LoggerEmpty", "logger is empty")
)
