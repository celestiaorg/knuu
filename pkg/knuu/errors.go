package knuu

import (
	"github.com/celestiaorg/knuu/pkg/errors"
)

type Error = errors.Error

var (
	ErrBitTwisterFailedToStart                   = errors.New("BitTwisterFailedToStart", "BitTwister failed to start")
	ErrCreatingInstance                          = errors.New("CreatingInstance", "error creating instance")
	ErrSettingImage                              = errors.New("SettingImage", "error setting image")
	ErrCommittingInstance                        = errors.New("CommittingInstance", "error committing instance")
	ErrSettingArgs                               = errors.New("SettingArgs", "error setting args")
	ErrSettingMemory                             = errors.New("SettingMemory", "error setting memory")
	ErrSettingCPU                                = errors.New("SettingCPU", "error setting cpu")
	ErrStartingInstance                          = errors.New("StartingInstance", "error starting instance")
	ErrWaitingInstanceIsRunning                  = errors.New("WaitingInstanceIsRunning", "error waiting for instance to be running")
	ErrPortNumberOutOfRange                      = errors.New("PortNumberOutOfRange", "port number '%d' is out of range")
	ErrDeployingService                          = errors.New("DeployingService", "error deploying service '%s'")
	ErrGettingService                            = errors.New("GettingService", "error getting service '%s'")
	ErrPatchingService                           = errors.New("PatchingService", "error patching service '%s'")
	ErrFailedToCreateServiceAccount              = errors.New("FailedToCreateServiceAccount", "failed to create service account")
	ErrFailedToCreateRole                        = errors.New("FailedToCreateRole", "failed to create role")
	ErrFailedToCreateRoleBinding                 = errors.New("FailedToCreateRoleBinding", "failed to create role binding")
	ErrFailedToDeployPod                         = errors.New("FailedToDeployPod", "failed to deploy pod")
	ErrFailedToDeletePod                         = errors.New("FailedToDeletePod", "failed to delete pod")
	ErrFailedToDeleteServiceAccount              = errors.New("FailedToDeleteServiceAccount", "failed to delete service account")
	ErrFailedToDeleteRole                        = errors.New("FailedToDeleteRole", "failed to delete role")
	ErrFailedToDeleteRoleBinding                 = errors.New("FailedToDeleteRoleBinding", "failed to delete role binding")
	ErrDeployingServiceForInstance               = errors.New("DeployingServiceForInstance", "error deploying service for instance '%s'")
	ErrPatchingServiceForInstance                = errors.New("PatchingServiceForInstance", "error patching service for instance '%s'")
	ErrFailedToOpenFile                          = errors.New("FailedToOpenFile", "failed to open file")
	ErrFailedToReadFile                          = errors.New("FailedToReadFile", "failed to read file")
	ErrFailedToCreateConfigMap                   = errors.New("FailedToCreateConfigMap", "failed to create configmap")
	ErrFailedToDeleteConfigMap                   = errors.New("FailedToDeleteConfigMap", "failed to delete configmap")
	ErrFailedToDeployOrPatchService              = errors.New("FailedToDeployOrPatchService", "failed to deploy or patch service")
	ErrDeployingServiceForSidecar                = errors.New("DeployingServiceForSidecar", "error deploying service for sidecar '%s' of instance '%s', a sidecar cannot have a service")
	ErrPatchingServiceForSidecar                 = errors.New("PatchingServiceForSidecar", "error patching service for sidecar '%s' of instance '%s', a sidecar cannot have a service")
	ErrDeployingVolumeForInstance                = errors.New("DeployingVolumeForInstance", "error deploying volume for instance '%s'")
	ErrDeployingFilesForInstance                 = errors.New("DeployingFilesForInstance", "error deploying files for instance '%s'")
	ErrDestroyingVolumeForInstance               = errors.New("DestroyingVolumeForInstance", "error destroying volume for instance '%s'")
	ErrDestroyingFilesForInstance                = errors.New("DestroyingFilesForInstance", "error destroying files for instance '%s'")
	ErrDestroyingServiceForInstance              = errors.New("DestroyingServiceForInstance", "error destroying service for instance '%s'")
	ErrCheckingNetworkStatusForInstance          = errors.New("CheckingNetworkStatusForInstance", "error checking network status for instance '%s'")
	ErrEnablingNetworkForInstance                = errors.New("EnablingNetworkForInstance", "error enabling network for instance '%s'")
	ErrGeneratingUUID                            = errors.New("GeneratingUUID", "error generating UUID")
	ErrGettingFreePort                           = errors.New("GettingFreePort", "error getting free port")
	ErrSrcMustBeSet                              = errors.New("SrcMustBeSet", "src must be set")
	ErrDestMustBeSet                             = errors.New("DestMustBeSet", "dest must be set")
	ErrChownMustBeSet                            = errors.New("ChownMustBeSet", "chown must be set")
	ErrChownMustBeInFormatUserGroup              = errors.New("ChownMustBeInFormatUserGroup", "chown must be in format 'user:group'")
	ErrAddingFileToInstance                      = errors.New("AddingFileToInstance", "error adding file '%s' to instance '%s'")
	ErrReplacingPod                              = errors.New("ReplacingPod", "error replacing pod")
	ErrApplyingFunctionToInstance                = errors.New("ApplyingFunctionToInstance", "error applying function to instance '%s'")
	ErrSettingNotAllowed                         = errors.New("SettingNotAllowed", "setting %s is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'")
	ErrCreatingOtelCollectorInstance             = errors.New("CreatingOtelCollectorInstance", "error creating otel collector instance '%s'")
	ErrSettingBitTwisterImage                    = errors.New("SettingBitTwisterImage", "error setting image for bit-twister instance")
	ErrAddingBitTwisterPort                      = errors.New("AddingBitTwisterPort", "error adding BitTwister port")
	ErrGettingInstanceIP                         = errors.New("GettingInstanceIP", "error getting IP of instance '%s'")
	ErrCommittingBitTwisterInstance              = errors.New("CommittingBitTwisterInstance", "error committing bit-twister instance")
	ErrSettingBitTwisterEnv                      = errors.New("SettingBitTwisterEnv", "error setting environment variable for bit-twister instance")
	ErrCreatingBitTwisterInstance                = errors.New("CreatingBitTwisterInstance", "error creating bit-twister instance '%s'")
	ErrSettingBitTwisterPrivileged               = errors.New("SettingBitTwisterPrivileged", "error setting privileged for bit-twister instance '%s'")
	ErrAddingBitTwisterCapability                = errors.New("AddingBitTwisterCapability", "error adding capability for bit-twister instance '%s'")
	ErrAddingBitTwisterSidecar                   = errors.New("AddingBitTwisterSidecar", "error adding bit-twister sidecar to instance '%s'")
	ErrCreatingOtelAgentInstance                 = errors.New("CreatingOtelAgentInstance", "error creating otel-agent instance")
	ErrSettingOtelAgentImage                     = errors.New("SettingOtelAgentImage", "error setting image for otel-agent instance")
	ErrAddingOtelAgentPort                       = errors.New("AddingOtelAgentPort", "error adding port for otel-agent instance")
	ErrSettingOtelAgentCPU                       = errors.New("SettingOtelAgentCPU", "error setting CPU for otel-agent instance")
	ErrSettingOtelAgentMemory                    = errors.New("SettingOtelAgentMemory", "error setting memory for otel-agent instance")
	ErrCommittingOtelAgentInstance               = errors.New("CommittingOtelAgentInstance", "error committing otel-agent instance")
	ErrMarshalingYAML                            = errors.New("MarshalingYAML", "error marshaling YAML")
	ErrAddingOtelAgentConfigFile                 = errors.New("AddingOtelAgentConfigFile", "error adding otel-agent config file")
	ErrSettingOtelAgentCommand                   = errors.New("SettingOtelAgentCommand", "error setting command for otel-agent instance")
	ErrCreatingPoolNotAllowed                    = errors.New("CreatingPoolNotAllowed", "creating a pool is only allowed in state 'Committed' or 'Destroyed'. Current state is '%s'")
	ErrGeneratingK8sName                         = errors.New("GeneratingK8sName", "error generating k8s name for instance '%s'")
	ErrEnablingBitTwister                        = errors.New("EnablingBitTwister", "enabling BitTwister is not allowed in state 'Started'")
	ErrSettingImageNotAllowed                    = errors.New("SettingImageNotAllowed", "setting image is only allowed in state 'None' and 'Started'. Current state is '%s'")
	ErrCreatingBuilder                           = errors.New("CreatingBuilder", "error creating builder")
	ErrSettingImageNotAllowedForSidecarsStarted  = errors.New("SettingImageNotAllowedForSidecarsStarted", "setting image is not allowed for sidecars when in state 'Started'")
	ErrSettingGitRepo                            = errors.New("SettingGitRepo", "setting git repo is only allowed in state 'None'. Current state is '%s'")
	ErrGettingBuildContext                       = errors.New("GettingBuildContext", "error getting build context")
	ErrGettingImageName                          = errors.New("GettingImageName", "error getting image name")
	ErrSettingImageNotAllowedForSidecars         = errors.New("SettingImageNotAllowedForSidecars", "setting image is not allowed for sidecars")
	ErrSettingCommand                            = errors.New("SettingCommand", "setting command is only allowed in state 'Preparing' or 'Committed'. Current state is '%s")
	ErrSettingArgsNotAllowed                     = errors.New("SettingArgsNotAllowed", "setting args is only allowed in state 'Preparing' or 'Committed'. Current state is '%s")
	ErrAddingPortNotAllowed                      = errors.New("AddingPortNotAllowed", "adding port is only allowed in state 'Preparing' or 'Committed'. Current state is '%s")
	ErrPortAlreadyRegistered                     = errors.New("PortAlreadyRegistered", "TCP port '%d' is already in registered")
	ErrRandomPortForwardingNotAllowed            = errors.New("RandomPortForwardingNotAllowed", "random port forwarding is only allowed in state 'Started'. Current state is '%s")
	ErrPortNotRegistered                         = errors.New("PortNotRegistered", "TCP port '%d' is not registered")
	ErrGettingPodFromReplicaSet                  = errors.New("GettingPodFromReplicaSet", "error getting pod from replicaset '%s'")
	ErrForwardingPort                            = errors.New("ForwardingPort", "error forwarding port after %d retries")
	ErrUDPPortAlreadyRegistered                  = errors.New("UDPPortAlreadyRegistered", "UDP port '%d' is already in registered")
	ErrExecutingCommandNotAllowed                = errors.New("ExecutingCommandNotAllowed", "executing command is only allowed in state 'Preparing' or 'Started'. Current state is '%s")
	ErrExecutingCommandInInstance                = errors.New("ExecutingCommandInInstance", "error executing command '%s' in instance '%s'")
	ErrExecutingCommandInSidecar                 = errors.New("ExecutingCommandInSidecar", "error executing command '%s' in sidecar '%s' of instance '%s'")
	ErrAddingFileNotAllowed                      = errors.New("AddingFileNotAllowed", "adding file is only allowed in state 'Preparing' or 'Committed'. Current state is '%s")
	ErrSrcDoesNotExist                           = errors.New("SrcDoesNotExist", "src '%s' does not exist")
	ErrCreatingDirectory                         = errors.New("CreatingDirectory", "error creating directory")
	ErrFailedToCreateDestFile                    = errors.New("FailedToCreateDestFile", "failed to create destination file '%s'")
	ErrFailedToOpenSrcFile                       = errors.New("FailedToOpenSrcFile", "failed to open source file '%s'")
	ErrFailedToCopyFile                          = errors.New("FailedToCopyFile", "failed to copy from source '%s' to destination '%s'")
	ErrSrcDoesNotExistOrIsDirectory              = errors.New("SrcDoesNotExistOrIsDirectory", "src '%s' does not exist or is a directory")
	ErrInvalidFormat                             = errors.New("InvalidFormat", "invalid format")
	ErrFailedToConvertToInt64                    = errors.New("FailedToConvertToInt64", "failed to convert to int64")
	ErrAllFilesMustHaveSameGroup                 = errors.New("AllFilesMustHaveSameGroup", "all files must have the same group")
	ErrAddingFolderNotAllowed                    = errors.New("AddingFolderNotAllowed", "adding folder is only allowed in state 'Preparing' or 'Committed'. Current state is '%s")
	ErrSrcDoesNotExistOrIsNotDirectory           = errors.New("SrcDoesNotExistOrIsNotDirectory", "src '%s' does not exist or is not a directory")
	ErrCopyingFolderToInstance                   = errors.New("CopyingFolderToInstance", "error copying folder '%s' to instance '%s")
	ErrSettingUserNotAllowed                     = errors.New("SettingUserNotAllowed", "setting user is only allowed in state 'Preparing'. Current state is '%s")
	ErrSettingUser                               = errors.New("SettingUser", "error setting user '%s' for instance '%s")
	ErrCommittingNotAllowed                      = errors.New("CommittingNotAllowed", "committing is only allowed in state 'Preparing'. Current state is '%s")
	ErrGettingImageRegistry                      = errors.New("GettingImageRegistry", "error getting image registry")
	ErrGeneratingImageHash                       = errors.New("GeneratingImageHash", "error generating image hash")
	ErrPushingImage                              = errors.New("PushingImage", "error pushing image for instance '%s'")
	ErrAddingVolumeNotAllowed                    = errors.New("AddingVolumeNotAllowed", "adding volume is only allowed in state 'Preparing' or 'Committed'. Current state is '%s")
	ErrSettingMemoryNotAllowed                   = errors.New("SettingMemoryNotAllowed", "setting memory is only allowed in state 'Preparing' or 'Committed'. Current state is '%s")
	ErrSettingCPUNotAllowed                      = errors.New("SettingCPUNotAllowed", "setting cpu is only allowed in state 'Preparing' or 'Committed'. Current state is '%s")
	ErrSettingEnvNotAllowed                      = errors.New("SettingEnvNotAllowed", "setting environment variable is only allowed in state 'Preparing' or 'Committed'. Current state is '%s")
	ErrGettingServiceForInstance                 = errors.New("GettingServiceForInstance", "error retrieving deployed service for instance '%s'")
	ErrGettingServiceIP                          = errors.New("GettingServiceIP", "IP address is not available for service '%s'")
	ErrGettingFileNotAllowed                     = errors.New("GettingFileNotAllowed", "getting file is only allowed in state 'Started', 'Preparing' or 'Committed'. Current state is '%s")
	ErrGettingFile                               = errors.New("GettingFile", "error getting file '%s' from instance '%s")
	ErrReadingFile                               = errors.New("ReadingFile", "error reading file '%s' from running instance '%s")
	ErrReadingFileNotAllowed                     = errors.New("ReadingFileNotAllowed", "reading file is only allowed in state 'Started'. Current state is '%s")
	ErrReadingFileFromInstance                   = errors.New("ReadingFileFromInstance", "error reading file '%s' from running instance '%s")
	ErrAddingPolicyRuleNotAllowed                = errors.New("AddingPolicyRuleNotAllowed", "adding policy rule is only allowed in state 'Preparing' or 'Committed'. Current state is '%s")
	ErrSettingProbeNotAllowed                    = errors.New("SettingProbeNotAllowed", "setting probe is only allowed in state 'Preparing' or 'Committed'. Current state is '%s")
	ErrAddingSidecarNotAllowed                   = errors.New("AddingSidecarNotAllowed", "adding sidecar is only allowed in state 'Preparing' or 'Committed'. Current state is '%s")
	ErrSidecarIsNil                              = errors.New("SidecarIsNil", "sidecar is nil")
	ErrSidecarCannotBeSameInstance               = errors.New("SidecarCannotBeSameInstance", "sidecar cannot be the same instance")
	ErrSidecarNotCommitted                       = errors.New("SidecarNotCommitted", "sidecar '%s' is not in state 'Committed'")
	ErrSidecarCannotHaveSidecar                  = errors.New("SidecarCannotHaveSidecar", "sidecar '%s' cannot have a sidecar")
	ErrSidecarAlreadySidecar                     = errors.New("SidecarAlreadySidecar", "sidecar '%s' is already a sidecar")
	ErrSettingPrivilegedNotAllowed               = errors.New("SettingPrivilegedNotAllowed", "setting privileged is only allowed in state 'Preparing' or 'Committed'. Current state is '%s")
	ErrAddingCapabilityNotAllowed                = errors.New("AddingCapabilityNotAllowed", "adding capability is only allowed in state 'Preparing' or 'Committed'. Current state is '%s")
	ErrAddingCapabilitiesNotAllowed              = errors.New("AddingCapabilitiesNotAllowed", "adding capabilities is only allowed in state 'Preparing' or 'Committed'. Current state is '%s")
	ErrStartingNotAllowed                        = errors.New("StartingNotAllowed", "starting is only allowed in state 'Committed' or 'Stopped'. Current state of sidecar '%s' is '%s'")
	ErrStartingNotAllowedForSidecar              = errors.New("StartingNotAllowedForSidecar", "starting is only allowed in state 'Committed' or 'Stopped'. Current state of sidecar '%s' is '%s")
	ErrStartingSidecarNotAllowed                 = errors.New("StartingSidecarNotAllowed", "starting a sidecar is not allowed")
	ErrAddingOtelCollectorSidecar                = errors.New("AddingOtelCollectorSidecar", "error adding OpenTelemetry collector sidecar for instance '%s'")
	ErrAddingNetworkSidecar                      = errors.New("AddingNetworkSidecar", "error adding network sidecar for instance '%s'")
	ErrDeployingResourcesForInstance             = errors.New("DeployingResourcesForInstance", "error deploying resources for instance '%s'")
	ErrDeployingResourcesForSidecars             = errors.New("DeployingResourcesForSidecars", "error deploying resources for sidecars of instance '%s'")
	ErrDeployingPodForInstance                   = errors.New("DeployingPodForInstance", "error deploying pod for instance '%s'")
	ErrWaitingForInstanceRunning                 = errors.New("WaitingForInstanceRunning", "error waiting for instance '%s' to be running")
	ErrCheckingIfInstanceRunningNotAllowed       = errors.New("CheckingIfInstanceRunningNotAllowed", "checking if instance is running is only allowed in state 'Started'. Current state is '%s")
	ErrWaitingForInstanceNotAllowed              = errors.New("WaitingForInstanceNotAllowed", "waiting for instance is only allowed in state 'Started'. Current state is '%s")
	ErrWaitingForInstanceTimeout                 = errors.New("WaitingForInstanceTimeout", "timeout while waiting for instance '%s' to be running")
	ErrCheckingIfInstanceRunning                 = errors.New("CheckingIfInstanceRunning", "error checking if instance '%s' is running")
	ErrDisablingNetworkNotAllowed                = errors.New("DisablingNetworkNotAllowed", "disabling network is only allowed in state 'Started'. Current state is '%s")
	ErrDisablingNetwork                          = errors.New("DisablingNetwork", "error disabling network for instance '%s'")
	ErrSettingBandwidthLimitNotAllowed           = errors.New("SettingBandwidthLimitNotAllowed", "setting bandwidth limit is only allowed in state 'Started'. Current state is '%s")
	ErrSettingBandwidthLimitNotAllowedBitTwister = errors.New("SettingBandwidthLimitNotAllowedBitTwister", "setting bandwidth limit is only allowed if BitTwister is enabled")
	ErrStoppingBandwidthLimit                    = errors.New("StoppingBandwidthLimit", "error stopping bandwidth limit for instance '%s'")
	ErrSettingBandwidthLimit                     = errors.New("SettingBandwidthLimit", "error setting bandwidth limit for instance '%s'")
	ErrSettingLatencyJitterNotAllowed            = errors.New("SettingLatencyJitterNotAllowed", "setting latency/jitter is only allowed in state 'Started'. Current state is '%s")
	ErrSettingLatencyJitterNotAllowedBitTwister  = errors.New("SettingLatencyJitterNotAllowedBitTwister", "setting latency/jitter is only allowed if BitTwister is enabled")
	ErrStoppingLatencyJitter                     = errors.New("StoppingLatencyJitter", "error stopping latency/jitter for instance '%s'")
	ErrSettingLatencyJitter                      = errors.New("SettingLatencyJitter", "error setting latency/jitter for instance '%s'")
	ErrSettingPacketLossNotAllowed               = errors.New("SettingPacketLossNotAllowed", "setting packetloss is only allowed in state 'Started'. Current state is '%s")
	ErrSettingPacketLossNotAllowedBitTwister     = errors.New("SettingPacketLossNotAllowedBitTwister", "setting packetloss is only allowed if BitTwister is enabled")
	ErrStoppingPacketLoss                        = errors.New("StoppingPacketLoss", "error stopping packetloss for instance '%s'")
	ErrSettingPacketLoss                         = errors.New("SettingPacketLoss", "error setting packetloss for instance '%s'")
	ErrEnablingNetworkNotAllowed                 = errors.New("EnablingNetworkNotAllowed", "enabling network is only allowed in state 'Started'. Current state is '%s")
	ErrEnablingNetwork                           = errors.New("EnablingNetwork", "error enabling network for instance '%s'")
	ErrCheckingIfNetworkDisabledNotAllowed       = errors.New("CheckingIfNetworkDisabledNotAllowed", "checking if network is disabled is only allowed in state 'Started'. Current state is '%s")
	ErrWaitingForInstanceStoppedNotAllowed       = errors.New("WaitingForInstanceStoppedNotAllowed", "waiting for instance is only allowed in state 'Stopped'. Current state is '%s")
	ErrCheckingIfInstanceStopped                 = errors.New("CheckingIfInstanceStopped", "error checking if instance '%s' is running")
	ErrStoppingNotAllowed                        = errors.New("StoppingNotAllowed", "stopping is only allowed in state 'Started'. Current state is '%s")
	ErrDestroyingNotAllowed                      = errors.New("DestroyingNotAllowed", "destroying is only allowed in state 'Started' or 'Destroyed'. Current state is '%s")
	ErrDestroyingPod                             = errors.New("DestroyingPod", "error destroying pod for instance '%s'")
	ErrDestroyingResourcesForInstance            = errors.New("DestroyingResourcesForInstance", "error destroying resources for instance '%s'")
	ErrDestroyingResourcesForSidecars            = errors.New("DestroyingResourcesForSidecars", "error destroying resources for sidecars of instance '%s'")
	ErrCloningNotAllowed                         = errors.New("CloningNotAllowed", "cloning is only allowed in state 'Committed'. Current state is '%s")
	ErrCloningNotAllowedForSidecar               = errors.New("CloningNotAllowedForSidecar", "cloning is only allowed in state 'Committed'. Current state is '%s")
	ErrGeneratingK8sNameForSidecar               = errors.New("GeneratingK8sNameForSidecar", "error generating k8s name for instance '%s'")
	ErrCannotInitializeKnuuWithEmptyScope        = errors.New("CannotInitializeKnuuWithEmptyScope", "cannot initialize knuu with empty scope")
	ErrCannotInitializeK8s                       = errors.New("CannotInitializeK8s", "cannot initialize k8s")
	ErrCreatingNamespace                         = errors.New("CreatingNamespace", "creating namespace %s")
	ErrCannotParseTimeout                        = errors.New("CannotParseTimeout", "cannot parse timeout")
	ErrCannotHandleTimeout                       = errors.New("CannotHandleTimeout", "cannot handle timeout")
	ErrInvalidKnuuBuilder                        = errors.New("InvalidKnuuBuilder", "invalid KNUU_BUILDER, available [kubernetes, docker], value used: %s")
	ErrCannotCreateInstance                      = errors.New("CannotCreateInstance", "cannot create instance")
	ErrCannotSetImage                            = errors.New("CannotSetImage", "cannot set image")
	ErrCannotCommitInstance                      = errors.New("CannotCommitInstance", "cannot commit instance")
	ErrCannotSetCommand                          = errors.New("CannotSetCommand", "cannot set command")
	ErrCannotAddPolicyRule                       = errors.New("CannotAddPolicyRule", "cannot add policy rule")
	ErrCannotStartInstance                       = errors.New("CannotStartInstance", "cannot start instance")
	ErrMinioNotInitialized                       = errors.New("MinioNotInitialized", "minio not initialized")
	ErrGeneratingK8sNameForPreloader             = errors.New("GeneratingK8sNameForPreloader", "error generating k8s name for preloader")
	ErrCannotLoadEnv                             = errors.New("CannotLoadEnv", "cannot load env")
	ErrMaximumVolumesExceeded                    = errors.New("MaximumVolumesExceeded", "maximum volumes exceeded for instance '%s'")
	ErrCustomResourceDefinitionDoesNotExist      = errors.New("CustomResourceDefinitionDoesNotExist", "custom resource definition %s does not exist")
	ErrFileIsNotSubFolderOfVolumes               = errors.New("FileIsNotSubFolderOfVolumes", "the file '%s' is not a sub folder of any added volume")
	ErrCannotInitializeKnuu                      = errors.New("CannotInitializeKnuu", "cannot initialize knuu")
	ErrCannotDeployTraefik                       = errors.New("CannotDeployTraefik", "cannot deploy Traefik")
	ErrGettingBitTwisterPath                     = errors.New("GettingBitTwisterPath", "error getting BitTwister path")
	ErrFailedToAddHostToTraefik                  = errors.New("FailedToAddHostToTraefik", "failed to add host to traefik")
	ErrParentInstanceIsNil                       = errors.New("ParentInstanceIsNil", "parent instance is nil for the sidecar '%s'")
	ErrFailedToGetIP                             = errors.New("FailedToGetIP", "failed to get IP for service %s")
	ErrNoParentInstance                          = errors.New("NoParentInstance", "no parent instance for the sidecar '%s'")
	ErrAddingToProxy                             = errors.New("AddingToTraefikProxy", "error adding '%s' to traefik proxy for service '%s'")
	ErrCannotGetTraefikEndpoint                  = errors.New("CannotGetTraefikEndpoint", "cannot get traefik endpoint")
	ErrGettingProxyURL                           = errors.New("GettingProxyURL", "error getting proxy URL for service '%s'")
	ErrTraefikAPINotAvailable                    = errors.New("TraefikAPINotAvailable", "traefik API is not available")
	ErrChaosMeshAPINotAvailable                  = errors.New("ChaosMeshAPINotAvailable", "chaos mesh API is not available")
)
