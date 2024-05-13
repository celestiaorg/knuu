package knuu

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
	ErrBitTwisterFailedToStart                   = &Error{Code: "BitTwisterFailedToStart", Message: "BitTwister failed to start"}
	ErrCreatingInstance                          = &Error{Code: "CreatingInstance", Message: "error creating instance"}
	ErrSettingImage                              = &Error{Code: "SettingImage", Message: "error setting image"}
	ErrCommittingInstance                        = &Error{Code: "CommittingInstance", Message: "error committing instance"}
	ErrSettingArgs                               = &Error{Code: "SettingArgs", Message: "error setting args"}
	ErrSettingMemory                             = &Error{Code: "SettingMemory", Message: "error setting memory"}
	ErrSettingCPU                                = &Error{Code: "SettingCPU", Message: "error setting cpu"}
	ErrStartingInstance                          = &Error{Code: "StartingInstance", Message: "error starting instance"}
	ErrWaitingInstanceIsRunning                  = &Error{Code: "WaitingInstanceIsRunning", Message: "error waiting for instance to be running"}
	ErrPortNumberOutOfRange                      = &Error{Code: "PortNumberOutOfRange", Message: "port number '%d' is out of range"}
	ErrDeployingService                          = &Error{Code: "DeployingService", Message: "error deploying service '%s'"}
	ErrGettingService                            = &Error{Code: "GettingService", Message: "error getting service '%s'"}
	ErrPatchingService                           = &Error{Code: "PatchingService", Message: "error patching service '%s'"}
	ErrFailedToCreateServiceAccount              = &Error{Code: "FailedToCreateServiceAccount", Message: "failed to create service account"}
	ErrFailedToCreateRole                        = &Error{Code: "FailedToCreateRole", Message: "failed to create role"}
	ErrFailedToCreateRoleBinding                 = &Error{Code: "FailedToCreateRoleBinding", Message: "failed to create role binding"}
	ErrFailedToDeployPod                         = &Error{Code: "FailedToDeployPod", Message: "failed to deploy pod"}
	ErrFailedToDeletePod                         = &Error{Code: "FailedToDeletePod", Message: "failed to delete pod"}
	ErrFailedToDeleteServiceAccount              = &Error{Code: "FailedToDeleteServiceAccount", Message: "failed to delete service account"}
	ErrFailedToDeleteRole                        = &Error{Code: "FailedToDeleteRole", Message: "failed to delete role"}
	ErrFailedToDeleteRoleBinding                 = &Error{Code: "FailedToDeleteRoleBinding", Message: "failed to delete role binding"}
	ErrDeployingServiceForInstance               = &Error{Code: "DeployingServiceForInstance", Message: "error deploying service for instance '%s'"}
	ErrPatchingServiceForInstance                = &Error{Code: "PatchingServiceForInstance", Message: "error patching service for instance '%s'"}
	ErrFailedToOpenFile                          = &Error{Code: "FailedToOpenFile", Message: "failed to open file"}
	ErrFailedToReadFile                          = &Error{Code: "FailedToReadFile", Message: "failed to read file"}
	ErrFailedToCreateConfigMap                   = &Error{Code: "FailedToCreateConfigMap", Message: "failed to create configmap"}
	ErrFailedToDeleteConfigMap                   = &Error{Code: "FailedToDeleteConfigMap", Message: "failed to delete configmap"}
	ErrFailedToDeployOrPatchService              = &Error{Code: "FailedToDeployOrPatchService", Message: "failed to deploy or patch service"}
	ErrDeployingVolumeForInstance                = &Error{Code: "DeployingVolumeForInstance", Message: "error deploying volume for instance '%s'"}
	ErrDeployingFilesForInstance                 = &Error{Code: "DeployingFilesForInstance", Message: "error deploying files for instance '%s'"}
	ErrDestroyingVolumeForInstance               = &Error{Code: "DestroyingVolumeForInstance", Message: "error destroying volume for instance '%s'"}
	ErrDestroyingFilesForInstance                = &Error{Code: "DestroyingFilesForInstance", Message: "error destroying files for instance '%s'"}
	ErrDestroyingServiceForInstance              = &Error{Code: "DestroyingServiceForInstance", Message: "error destroying service for instance '%s'"}
	ErrCheckingNetworkStatusForInstance          = &Error{Code: "CheckingNetworkStatusForInstance", Message: "error checking network status for instance '%s'"}
	ErrEnablingNetworkForInstance                = &Error{Code: "EnablingNetworkForInstance", Message: "error enabling network for instance '%s'"}
	ErrGeneratingUUID                            = &Error{Code: "GeneratingUUID", Message: "error generating UUID"}
	ErrGettingFreePort                           = &Error{Code: "GettingFreePort", Message: "error getting free port"}
	ErrSrcMustBeSet                              = &Error{Code: "SrcMustBeSet", Message: "src must be set"}
	ErrDestMustBeSet                             = &Error{Code: "DestMustBeSet", Message: "dest must be set"}
	ErrChownMustBeSet                            = &Error{Code: "ChownMustBeSet", Message: "chown must be set"}
	ErrChownMustBeInFormatUserGroup              = &Error{Code: "ChownMustBeInFormatUserGroup", Message: "chown must be in format 'user:group'"}
	ErrAddingFileToInstance                      = &Error{Code: "AddingFileToInstance", Message: "error adding file '%s' to instance '%s'"}
	ErrReplacingPod                              = &Error{Code: "ReplacingPod", Message: "error replacing pod"}
	ErrApplyingFunctionToInstance                = &Error{Code: "ApplyingFunctionToInstance", Message: "error applying function to instance '%s'"}
	ErrSettingNotAllowed                         = &Error{Code: "SettingNotAllowed", Message: "setting %s is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'"}
	ErrCreatingOtelCollectorInstance             = &Error{Code: "CreatingOtelCollectorInstance", Message: "error creating otel collector instance '%s'"}
	ErrSettingBitTwisterImage                    = &Error{Code: "SettingBitTwisterImage", Message: "error setting image for bit-twister instance"}
	ErrAddingBitTwisterPort                      = &Error{Code: "AddingBitTwisterPort", Message: "error adding BitTwister port"}
	ErrGettingInstanceIP                         = &Error{Code: "GettingInstanceIP", Message: "error getting IP of instance '%s'"}
	ErrCommittingBitTwisterInstance              = &Error{Code: "CommittingBitTwisterInstance", Message: "error committing bit-twister instance"}
	ErrSettingBitTwisterEnv                      = &Error{Code: "SettingBitTwisterEnv", Message: "error setting environment variable for bit-twister instance"}
	ErrCreatingBitTwisterInstance                = &Error{Code: "CreatingBitTwisterInstance", Message: "error creating bit-twister instance '%s'"}
	ErrSettingBitTwisterPrivileged               = &Error{Code: "SettingBitTwisterPrivileged", Message: "error setting privileged for bit-twister instance '%s'"}
	ErrAddingBitTwisterCapability                = &Error{Code: "AddingBitTwisterCapability", Message: "error adding capability for bit-twister instance '%s'"}
	ErrAddingBitTwisterSidecar                   = &Error{Code: "AddingBitTwisterSidecar", Message: "error adding bit-twister sidecar to instance '%s'"}
	ErrCreatingOtelAgentInstance                 = &Error{Code: "CreatingOtelAgentInstance", Message: "error creating otel-agent instance"}
	ErrSettingOtelAgentImage                     = &Error{Code: "SettingOtelAgentImage", Message: "error setting image for otel-agent instance"}
	ErrAddingOtelAgentPort                       = &Error{Code: "AddingOtelAgentPort", Message: "error adding port for otel-agent instance"}
	ErrSettingOtelAgentCPU                       = &Error{Code: "SettingOtelAgentCPU", Message: "error setting CPU for otel-agent instance"}
	ErrSettingOtelAgentMemory                    = &Error{Code: "SettingOtelAgentMemory", Message: "error setting memory for otel-agent instance"}
	ErrCommittingOtelAgentInstance               = &Error{Code: "CommittingOtelAgentInstance", Message: "error committing otel-agent instance"}
	ErrMarshalingYAML                            = &Error{Code: "MarshalingYAML", Message: "error marshaling YAML"}
	ErrAddingOtelAgentConfigFile                 = &Error{Code: "AddingOtelAgentConfigFile", Message: "error adding otel-agent config file"}
	ErrSettingOtelAgentCommand                   = &Error{Code: "SettingOtelAgentCommand", Message: "error setting command for otel-agent instance"}
	ErrCreatingPoolNotAllowed                    = &Error{Code: "CreatingPoolNotAllowed", Message: "creating a pool is only allowed in state 'Committed' or 'Destroyed'. Current state is '%s'"}
	ErrGeneratingK8sName                         = &Error{Code: "GeneratingK8sName", Message: "error generating k8s name for instance '%s'"}
	ErrEnablingBitTwister                        = &Error{Code: "EnablingBitTwister", Message: "enabling BitTwister is not allowed in state 'Started'"}
	ErrSettingImageNotAllowed                    = &Error{Code: "SettingImageNotAllowed", Message: "setting image is only allowed in state 'None' and 'Started'. Current state is '%s'"}
	ErrCreatingBuilder                           = &Error{Code: "CreatingBuilder", Message: "error creating builder"}
	ErrSettingImageNotAllowedForSidecarsStarted  = &Error{Code: "SettingImageNotAllowedForSidecarsStarted", Message: "setting image is not allowed for sidecars when in state 'Started'"}
	ErrSettingGitRepo                            = &Error{Code: "SettingGitRepo", Message: "setting git repo is only allowed in state 'None'. Current state is '%s'"}
	ErrGettingBuildContext                       = &Error{Code: "GettingBuildContext", Message: "error getting build context"}
	ErrGettingImageName                          = &Error{Code: "GettingImageName", Message: "error getting image name"}
	ErrSettingImageNotAllowedForSidecars         = &Error{Code: "SettingImageNotAllowedForSidecars", Message: "setting image is not allowed for sidecars"}
	ErrSettingCommand                            = &Error{Code: "SettingCommand", Message: "setting command is only allowed in state 'Preparing' or 'Committed'. Current state is '%s"}
	ErrSettingArgsNotAllowed                     = &Error{Code: "SettingArgsNotAllowed", Message: "setting args is only allowed in state 'Preparing' or 'Committed'. Current state is '%s"}
	ErrAddingPortNotAllowed                      = &Error{Code: "AddingPortNotAllowed", Message: "adding port is only allowed in state 'Preparing' or 'Committed'. Current state is '%s"}
	ErrPortAlreadyRegistered                     = &Error{Code: "PortAlreadyRegistered", Message: "TCP port '%d' is already in registered"}
	ErrRandomPortForwardingNotAllowed            = &Error{Code: "RandomPortForwardingNotAllowed", Message: "random port forwarding is only allowed in state 'Started'. Current state is '%s"}
	ErrPortNotRegistered                         = &Error{Code: "PortNotRegistered", Message: "TCP port '%d' is not registered"}
	ErrGettingPodFromReplicaSet                  = &Error{Code: "GettingPodFromReplicaSet", Message: "error getting pod from replicaset '%s'"}
	ErrForwardingPort                            = &Error{Code: "ForwardingPort", Message: "error forwarding port after %d retries"}
	ErrUDPPortAlreadyRegistered                  = &Error{Code: "UDPPortAlreadyRegistered", Message: "UDP port '%d' is already in registered"}
	ErrExecutingCommandNotAllowed                = &Error{Code: "ExecutingCommandNotAllowed", Message: "executing command is only allowed in state 'Preparing' or 'Started'. Current state is '%s"}
	ErrExecutingCommandInInstance                = &Error{Code: "ExecutingCommandInInstance", Message: "error executing command '%s' in instance '%s'"}
	ErrExecutingCommandInSidecar                 = &Error{Code: "ExecutingCommandInSidecar", Message: "error executing command '%s' in sidecar '%s' of instance '%s'"}
	ErrAddingFileNotAllowed                      = &Error{Code: "AddingFileNotAllowed", Message: "adding file is only allowed in state 'Preparing' or 'Committed'. Current state is '%s"}
	ErrSrcDoesNotExist                           = &Error{Code: "SrcDoesNotExist", Message: "src '%s' does not exist"}
	ErrCreatingDirectory                         = &Error{Code: "CreatingDirectory", Message: "error creating directory"}
	ErrFailedToCreateDestFile                    = &Error{Code: "FailedToCreateDestFile", Message: "failed to create destination file '%s'"}
	ErrFailedToOpenSrcFile                       = &Error{Code: "FailedToOpenSrcFile", Message: "failed to open source file '%s'"}
	ErrFailedToCopyFile                          = &Error{Code: "FailedToCopyFile", Message: "failed to copy from source '%s' to destination '%s'"}
	ErrSrcDoesNotExistOrIsDirectory              = &Error{Code: "SrcDoesNotExistOrIsDirectory", Message: "src '%s' does not exist or is a directory"}
	ErrInvalidFormat                             = &Error{Code: "InvalidFormat", Message: "invalid format"}
	ErrFailedToConvertToInt64                    = &Error{Code: "FailedToConvertToInt64", Message: "failed to convert to int64"}
	ErrAllFilesMustHaveSameGroup                 = &Error{Code: "AllFilesMustHaveSameGroup", Message: "all files must have the same group"}
	ErrAddingFolderNotAllowed                    = &Error{Code: "AddingFolderNotAllowed", Message: "adding folder is only allowed in state 'Preparing' or 'Committed'. Current state is '%s"}
	ErrSrcDoesNotExistOrIsNotDirectory           = &Error{Code: "SrcDoesNotExistOrIsNotDirectory", Message: "src '%s' does not exist or is not a directory"}
	ErrCopyingFolderToInstance                   = &Error{Code: "CopyingFolderToInstance", Message: "error copying folder '%s' to instance '%s"}
	ErrSettingUserNotAllowed                     = &Error{Code: "SettingUserNotAllowed", Message: "setting user is only allowed in state 'Preparing'. Current state is '%s"}
	ErrSettingUser                               = &Error{Code: "SettingUser", Message: "error setting user '%s' for instance '%s"}
	ErrCommittingNotAllowed                      = &Error{Code: "CommittingNotAllowed", Message: "committing is only allowed in state 'Preparing'. Current state is '%s"}
	ErrGettingImageRegistry                      = &Error{Code: "GettingImageRegistry", Message: "error getting image registry"}
	ErrGeneratingImageHash                       = &Error{Code: "GeneratingImageHash", Message: "error generating image hash"}
	ErrPushingImage                              = &Error{Code: "PushingImage", Message: "error pushing image for instance '%s'"}
	ErrAddingVolumeNotAllowed                    = &Error{Code: "AddingVolumeNotAllowed", Message: "adding volume is only allowed in state 'Preparing' or 'Committed'. Current state is '%s"}
	ErrSettingMemoryNotAllowed                   = &Error{Code: "SettingMemoryNotAllowed", Message: "setting memory is only allowed in state 'Preparing' or 'Committed'. Current state is '%s"}
	ErrSettingCPUNotAllowed                      = &Error{Code: "SettingCPUNotAllowed", Message: "setting cpu is only allowed in state 'Preparing' or 'Committed'. Current state is '%s"}
	ErrSettingEnvNotAllowed                      = &Error{Code: "SettingEnvNotAllowed", Message: "setting environment variable is only allowed in state 'Preparing' or 'Committed'. Current state is '%s"}
	ErrGettingServiceForInstance                 = &Error{Code: "GettingServiceForInstance", Message: "error retrieving deployed service for instance '%s'"}
	ErrGettingServiceIP                          = &Error{Code: "GettingServiceIP", Message: "IP address is not available for service '%s'"}
	ErrGettingFileNotAllowed                     = &Error{Code: "GettingFileNotAllowed", Message: "getting file is only allowed in state 'Started', 'Preparing' or 'Committed'. Current state is '%s"}
	ErrGettingFile                               = &Error{Code: "GettingFile", Message: "error getting file '%s' from instance '%s"}
	ErrReadingFile                               = &Error{Code: "ReadingFile", Message: "error reading file '%s' from running instance '%s"}
	ErrReadingFileNotAllowed                     = &Error{Code: "ReadingFileNotAllowed", Message: "reading file is only allowed in state 'Started'. Current state is '%s"}
	ErrReadingFileFromInstance                   = &Error{Code: "ReadingFileFromInstance", Message: "error reading file '%s' from running instance '%s"}
	ErrAddingPolicyRuleNotAllowed                = &Error{Code: "AddingPolicyRuleNotAllowed", Message: "adding policy rule is only allowed in state 'Preparing' or 'Committed'. Current state is '%s"}
	ErrSettingProbeNotAllowed                    = &Error{Code: "SettingProbeNotAllowed", Message: "setting probe is only allowed in state 'Preparing' or 'Committed'. Current state is '%s"}
	ErrAddingSidecarNotAllowed                   = &Error{Code: "AddingSidecarNotAllowed", Message: "adding sidecar is only allowed in state 'Preparing' or 'Committed'. Current state is '%s"}
	ErrSidecarIsNil                              = &Error{Code: "SidecarIsNil", Message: "sidecar is nil"}
	ErrSidecarCannotBeSameInstance               = &Error{Code: "SidecarCannotBeSameInstance", Message: "sidecar cannot be the same instance"}
	ErrSidecarNotCommitted                       = &Error{Code: "SidecarNotCommitted", Message: "sidecar '%s' is not in state 'Committed'"}
	ErrSidecarCannotHaveSidecar                  = &Error{Code: "SidecarCannotHaveSidecar", Message: "sidecar '%s' cannot have a sidecar"}
	ErrSidecarAlreadySidecar                     = &Error{Code: "SidecarAlreadySidecar", Message: "sidecar '%s' is already a sidecar"}
	ErrSettingPrivilegedNotAllowed               = &Error{Code: "SettingPrivilegedNotAllowed", Message: "setting privileged is only allowed in state 'Preparing' or 'Committed'. Current state is '%s"}
	ErrAddingCapabilityNotAllowed                = &Error{Code: "AddingCapabilityNotAllowed", Message: "adding capability is only allowed in state 'Preparing' or 'Committed'. Current state is '%s"}
	ErrAddingCapabilitiesNotAllowed              = &Error{Code: "AddingCapabilitiesNotAllowed", Message: "adding capabilities is only allowed in state 'Preparing' or 'Committed'. Current state is '%s"}
	ErrStartingNotAllowed                        = &Error{Code: "StartingNotAllowed", Message: "starting is only allowed in state 'Committed' or 'Stopped'. Current state of sidecar '%s' is '%s'"}
	ErrStartingNotAllowedForSidecar              = &Error{Code: "StartingNotAllowedForSidecar", Message: "starting is only allowed in state 'Committed' or 'Stopped'. Current state of sidecar '%s' is '%s"}
	ErrStartingSidecarNotAllowed                 = &Error{Code: "StartingSidecarNotAllowed", Message: "starting a sidecar is not allowed"}
	ErrAddingOtelCollectorSidecar                = &Error{Code: "AddingOtelCollectorSidecar", Message: "error adding OpenTelemetry collector sidecar for instance '%s'"}
	ErrAddingNetworkSidecar                      = &Error{Code: "AddingNetworkSidecar", Message: "error adding network sidecar for instance '%s'"}
	ErrDeployingResourcesForInstance             = &Error{Code: "DeployingResourcesForInstance", Message: "error deploying resources for instance '%s'"}
	ErrDeployingResourcesForSidecars             = &Error{Code: "DeployingResourcesForSidecars", Message: "error deploying resources for sidecars of instance '%s'"}
	ErrDeployingPodForInstance                   = &Error{Code: "DeployingPodForInstance", Message: "error deploying pod for instance '%s'"}
	ErrWaitingForInstanceRunning                 = &Error{Code: "WaitingForInstanceRunning", Message: "error waiting for instance '%s' to be running"}
	ErrCheckingIfInstanceRunningNotAllowed       = &Error{Code: "CheckingIfInstanceRunningNotAllowed", Message: "checking if instance is running is only allowed in state 'Started'. Current state is '%s"}
	ErrWaitingForInstanceNotAllowed              = &Error{Code: "WaitingForInstanceNotAllowed", Message: "waiting for instance is only allowed in state 'Started'. Current state is '%s"}
	ErrWaitingForInstanceTimeout                 = &Error{Code: "WaitingForInstanceTimeout", Message: "timeout while waiting for instance '%s' to be running"}
	ErrCheckingIfInstanceRunning                 = &Error{Code: "CheckingIfInstanceRunning", Message: "error checking if instance '%s' is running"}
	ErrDisablingNetworkNotAllowed                = &Error{Code: "DisablingNetworkNotAllowed", Message: "disabling network is only allowed in state 'Started'. Current state is '%s"}
	ErrDisablingNetwork                          = &Error{Code: "DisablingNetwork", Message: "error disabling network for instance '%s'"}
	ErrSettingBandwidthLimitNotAllowed           = &Error{Code: "SettingBandwidthLimitNotAllowed", Message: "setting bandwidth limit is only allowed in state 'Started'. Current state is '%s"}
	ErrSettingBandwidthLimitNotAllowedBitTwister = &Error{Code: "SettingBandwidthLimitNotAllowedBitTwister", Message: "setting bandwidth limit is only allowed if BitTwister is enabled"}
	ErrStoppingBandwidthLimit                    = &Error{Code: "StoppingBandwidthLimit", Message: "error stopping bandwidth limit for instance '%s'"}
	ErrSettingBandwidthLimit                     = &Error{Code: "SettingBandwidthLimit", Message: "error setting bandwidth limit for instance '%s'"}
	ErrSettingLatencyJitterNotAllowed            = &Error{Code: "SettingLatencyJitterNotAllowed", Message: "setting latency/jitter is only allowed in state 'Started'. Current state is '%s"}
	ErrSettingLatencyJitterNotAllowedBitTwister  = &Error{Code: "SettingLatencyJitterNotAllowedBitTwister", Message: "setting latency/jitter is only allowed if BitTwister is enabled"}
	ErrStoppingLatencyJitter                     = &Error{Code: "StoppingLatencyJitter", Message: "error stopping latency/jitter for instance '%s'"}
	ErrSettingLatencyJitter                      = &Error{Code: "SettingLatencyJitter", Message: "error setting latency/jitter for instance '%s'"}
	ErrSettingPacketLossNotAllowed               = &Error{Code: "SettingPacketLossNotAllowed", Message: "setting packetloss is only allowed in state 'Started'. Current state is '%s"}
	ErrSettingPacketLossNotAllowedBitTwister     = &Error{Code: "SettingPacketLossNotAllowedBitTwister", Message: "setting packetloss is only allowed if BitTwister is enabled"}
	ErrStoppingPacketLoss                        = &Error{Code: "StoppingPacketLoss", Message: "error stopping packetloss for instance '%s'"}
	ErrSettingPacketLoss                         = &Error{Code: "SettingPacketLoss", Message: "error setting packetloss for instance '%s'"}
	ErrEnablingNetworkNotAllowed                 = &Error{Code: "EnablingNetworkNotAllowed", Message: "enabling network is only allowed in state 'Started'. Current state is '%s"}
	ErrEnablingNetwork                           = &Error{Code: "EnablingNetwork", Message: "error enabling network for instance '%s'"}
	ErrCheckingIfNetworkDisabledNotAllowed       = &Error{Code: "CheckingIfNetworkDisabledNotAllowed", Message: "checking if network is disabled is only allowed in state 'Started'. Current state is '%s"}
	ErrWaitingForInstanceStoppedNotAllowed       = &Error{Code: "WaitingForInstanceStoppedNotAllowed", Message: "waiting for instance is only allowed in state 'Stopped'. Current state is '%s"}
	ErrCheckingIfInstanceStopped                 = &Error{Code: "CheckingIfInstanceStopped", Message: "error checking if instance '%s' is running"}
	ErrStoppingNotAllowed                        = &Error{Code: "StoppingNotAllowed", Message: "stopping is only allowed in state 'Started'. Current state is '%s"}
	ErrDestroyingNotAllowed                      = &Error{Code: "DestroyingNotAllowed", Message: "destroying is only allowed in state 'Started' or 'Destroyed'. Current state is '%s"}
	ErrDestroyingPod                             = &Error{Code: "DestroyingPod", Message: "error destroying pod for instance '%s'"}
	ErrDestroyingResourcesForInstance            = &Error{Code: "DestroyingResourcesForInstance", Message: "error destroying resources for instance '%s'"}
	ErrDestroyingResourcesForSidecars            = &Error{Code: "DestroyingResourcesForSidecars", Message: "error destroying resources for sidecars of instance '%s'"}
	ErrCloningNotAllowed                         = &Error{Code: "CloningNotAllowed", Message: "cloning is only allowed in state 'Committed'. Current state is '%s"}
	ErrCloningNotAllowedForSidecar               = &Error{Code: "CloningNotAllowedForSidecar", Message: "cloning is only allowed in state 'Committed'. Current state is '%s"}
	ErrGeneratingK8sNameForSidecar               = &Error{Code: "GeneratingK8sNameForSidecar", Message: "error generating k8s name for instance '%s'"}
	ErrCannotInitializeKnuuWithEmptyScope        = &Error{Code: "Cannot Initialize Knuu With Empty Scope", Message: "cannot initialize knuu with empty scope"}
	ErrCannotInitializeK8s                       = &Error{Code: "Cannot Initialize K8s", Message: "cannot initialize k8s"}
	ErrCreatingNamespace                         = &Error{Code: "CreatingNamespace", Message: "creating namespace %s"}
	ErrCannotParseTimeout                        = &Error{Code: "Cannot Parse Timeout", Message: "cannot parse timeout"}
	ErrCannotHandleTimeout                       = &Error{Code: "Cannot Handle Timeout", Message: "cannot handle timeout"}
	ErrInvalidKnuuBuilder                        = &Error{Code: "Invalid Knuu Builder", Message: "invalid KNUU_BUILDER, available [kubernetes, docker], value used: %s"}
	ErrCannotCreateInstance                      = &Error{Code: "Cannot Create Instance", Message: "cannot create instance"}
	ErrCannotSetImage                            = &Error{Code: "Cannot Set Image", Message: "cannot set image"}
	ErrCannotCommitInstance                      = &Error{Code: "Cannot Commit Instance", Message: "cannot commit instance"}
	ErrCannotSetCommand                          = &Error{Code: "Cannot Set Command", Message: "cannot set command"}
	ErrCannotAddPolicyRule                       = &Error{Code: "Cannot Add Policy Rule", Message: "cannot add policy rule"}
	ErrCannotStartInstance                       = &Error{Code: "Cannot Start Instance", Message: "cannot start instance"}
	ErrMinioNotInitialized                       = &Error{Code: "MinioNotInitialized", Message: "minio not initialized"}
	ErrGeneratingK8sNameForPreloader             = &Error{Code: "GeneratingK8sNameForPreloader", Message: "error generating k8s name for preloader"}
	ErrCannotLoadEnv                             = &Error{Code: "Cannot Load Env", Message: "cannot load env"}
	ErrMaximumVolumesExceeded                    = &Error{Code: "MaximumVolumesExceeded", Message: "maximum volumes exceeded for instance '%s'"}
)
