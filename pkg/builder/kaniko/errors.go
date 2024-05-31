package kaniko

import (
	"github.com/celestiaorg/knuu/pkg/errors"
)

type Error = errors.Error

var (
	ErrBuildFailed                      = errors.New("BuildFailed", "build failed")
	ErrBuildContextEmpty                = errors.New("BuildContextEmpty", "build context cannot be empty")
	ErrCleaningUp                       = errors.New("CleaningUp", "error cleaning up")
	ErrCreatingJob                      = errors.New("CreatingJob", "error creating Job")
	ErrDeletingJob                      = errors.New("DeletingJob", "error deleting Job")
	ErrDeletingPods                     = errors.New("DeletingPods", "error deleting Pods")
	ErrGeneratingUUID                   = errors.New("GeneratingUUID", "error generating UUID")
	ErrGettingContainerLogs             = errors.New("GettingContainerLogs", "error getting container logs")
	ErrGettingPodFromJob                = errors.New("GettingPodFromJob", "error getting Pod from Job")
	ErrListingJobs                      = errors.New("ListingJobs", "error listing Jobs")
	ErrListingPods                      = errors.New("ListingPods", "error listing Pods")
	ErrNoContainersFound                = errors.New("NoContainersFound", "no containers found")
	ErrNoPodsFound                      = errors.New("NoPodsFound", "no Pods found")
	ErrPreparingJob                     = errors.New("PreparingJob", "error preparing Job")
	ErrWaitingJobCompletion             = errors.New("WaitingJobCompletion", "error waiting for Job completion")
	ErrWatchingChannelCloseUnexpectedly = errors.New("WatchingChannelCloseUnexpectedly", "watch channel closed unexpectedly")
	ErrWatchingJob                      = errors.New("WatchingJob", "error watching Job")
	ErrContextCancelled                 = errors.New("ContextCancelled", "context cancelled")
	ErrMountingDir                      = errors.New("MountingDir", "error mounting directory")
	ErrMinioNotConfigured               = errors.New("MinioNotConfigured", "Minio service is not configured")
	ErrMinioDeploymentFailed            = errors.New("MinioDeploymentFailed", "Minio deployment failed")
	ErrDeletingMinioContent             = errors.New("DeletingMinioContent", "error deleting Minio content")
	ErrParsingQuantity                  = errors.New("ParsingQuantity", "error parsing quantity")
)
