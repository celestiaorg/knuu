package kaniko

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
	ErrBuildContextEmpty                = &Error{Code: "BuildContextEmpty", Message: "build context cannot be empty"}
	ErrCleaningUp                       = &Error{Code: "CleaningUp", Message: "error cleaning up"}
	ErrCreatingJob                      = &Error{Code: "CreatingJob", Message: "error creating Job"}
	ErrDeletingJob                      = &Error{Code: "DeletingJob", Message: "error deleting Job"}
	ErrDeletingPods                     = &Error{Code: "DeletingPods", Message: "error deleting Pods"}
	ErrGeneratingUUID                   = &Error{Code: "GeneratingUUID", Message: "error generating UUID"}
	ErrGettingContainerLogs             = &Error{Code: "GettingContainerLogs", Message: "error getting container logs"}
	ErrGettingPodFromJob                = &Error{Code: "GettingPodFromJob", Message: "error getting Pod from Job"}
	ErrListingJobs                      = &Error{Code: "ListingJobs", Message: "error listing Jobs"}
	ErrListingPods                      = &Error{Code: "ListingPods", Message: "error listing Pods"}
	ErrNoContainersFound                = &Error{Code: "NoContainersFound", Message: "no containers found"}
	ErrNoPodsFound                      = &Error{Code: "NoPodsFound", Message: "no Pods found"}
	ErrPreparingJob                     = &Error{Code: "PreparingJob", Message: "error preparing Job"}
	ErrWaitingJobCompletion             = &Error{Code: "WaitingJobCompletion", Message: "error waiting for Job completion"}
	ErrWatchingChannelCloseUnexpectedly = &Error{Code: "WatchingChannelCloseUnexpectedly", Message: "watch channel closed unexpectedly"}
	ErrWatchingJob                      = &Error{Code: "WatchingJob", Message: "error watching Job"}
	ErrContextCancelled                 = &Error{Code: "ContextCancelled", Message: "context cancelled"}
)
