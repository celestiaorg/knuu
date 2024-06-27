package instance

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/celestiaorg/bittwister/sdk"
	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/container"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/names"
	"github.com/celestiaorg/knuu/pkg/system"
)

// We need to retry here because the port forwarding might fail as getFreePortTCP() might not free the port fast enough
const (
	maxRetries    = 5
	retryInterval = 5 * time.Second
)

// ObsyConfig represents the configuration for the obsy sidecar
type ObsyConfig struct {
	// otelCollectorVersion is the version of the otel collector to use
	otelCollectorVersion string

	// prometheusEndpointPort is the port on which the prometheus server will be exposed
	prometheusEndpointPort int
	// prometheusEndpointJobName is the name of the prometheus job
	prometheusEndpointJobName string
	// prometheusEndpointScrapeInterval is the scrape interval for the prometheus job
	prometheusEndpointScrapeInterval string

	// jaegerGrpcPort is the port on which the jaeger grpc server is exposed
	jaegerGrpcPort int
	// jaegerThriftCompactPort is the port on which the jaeger thrift compact server is exposed
	jaegerThriftCompactPort int
	// jaegerThriftHttpPort is the port on which the jaeger thrift http server is exposed
	jaegerThriftHttpPort int
	// jaegerEndpoint is the endpoint of the jaeger collector where spans will be sent to
	jaegerEndpoint string

	// otlpPort is the port on which the otlp server is exposed
	otlpPort int
	// otlpEndpoint is the endpoint of the otlp collector where spans will be sent to
	otlpEndpoint string
	// otlpUsername is the username to use for the otlp collector
	otlpUsername string
	// otlpPassword is the password to use for the otlp collector
	otlpPassword string

	// prometheusExporterEndpoint is the endpoint of the prometheus exporter
	prometheusExporterEndpoint string

	// prometheusRemoteWriteExporterEndpoint is the endpoint of the prometheus remote write
	prometheusRemoteWriteExporterEndpoint string
}

// TsharkCollectorConfig represents the configuration for the tshark collector
type TsharkCollectorConfig struct {
	// VolumeSize is the size of the volume to use for the tshark collector
	VolumeSize resource.Quantity
	// S3AccessKey is the access key to use for the s3 server
	S3AccessKey string
	// S3SecretKey is the secret key to use for the s3 server
	S3SecretKey string
	// S3Region is the region of the s3 server
	S3Region string
	// S3Bucket is the bucket to use for the s3 server
	S3Bucket string
	// CreateBucket is the flag to create the bucket if it does not exist
	CreateBucket bool
	// S3KeyPrefix is the key prefix to use for the s3 server
	S3KeyPrefix string
	// S3Endpoint is the endpoint of the s3 server
	S3Endpoint string

	// UploadInterval is the interval at which the tshark collector will upload the pcap file to the s3 server
	UploadInterval time.Duration

	// IpFilter is the ip filter to use for the tshark collector
	// it trace the incoming/outgoing traffic from/to the specific ip
	// If not set, it will trace all the traffic
	IpFilter string

	// CompressFiles is the flag to compress the pcap files before pushing them to s3
	CompressFiles bool
}

// SecurityContext represents the security settings for a container
type SecurityContext struct {
	// Privileged indicates whether the container should be run in privileged mode
	privileged bool

	// CapabilitiesAdd is the list of capabilities to add to the container
	capabilitiesAdd []string
}

// Instance represents a instance
type Instance struct {
	system.SystemDependencies
	name                  string
	imageName             string
	k8sName               string
	state                 InstanceState
	instanceType          InstanceType
	kubernetesService     *v1.Service
	builderFactory        *container.BuilderFactory
	kubernetesReplicaSet  *appv1.ReplicaSet
	portsTCP              []int
	portsUDP              []int
	command               []string
	args                  []string
	env                   map[string]string
	volumes               []*k8s.Volume
	memoryRequest         resource.Quantity
	memoryLimit           resource.Quantity
	cpuRequest            resource.Quantity
	policyRules           []rbacv1.PolicyRule
	livenessProbe         *v1.Probe
	readinessProbe        *v1.Probe
	startupProbe          *v1.Probe
	files                 []*k8s.File
	isSidecar             bool
	parentInstance        *Instance
	sidecars              []*Instance
	fsGroup               int64
	obsyConfig            *ObsyConfig
	tsharkCollectorConfig *TsharkCollectorConfig
	securityContext       *SecurityContext
	BitTwister            *btConfig
}

func New(name string, sysDeps system.SystemDependencies) (*Instance, error) {
	k8sName, err := names.NewRandomK8(name)
	if err != nil {
		return nil, ErrGeneratingK8sName.WithParams(name).Wrap(err)
	}

	obsyConfig := &ObsyConfig{
		otelCollectorVersion:                  "0.83.0",
		otlpPort:                              0,
		prometheusEndpointPort:                0,
		prometheusEndpointJobName:             "",
		prometheusEndpointScrapeInterval:      "",
		jaegerGrpcPort:                        0,
		jaegerThriftCompactPort:               0,
		jaegerThriftHttpPort:                  0,
		otlpEndpoint:                          "",
		otlpUsername:                          "",
		otlpPassword:                          "",
		jaegerEndpoint:                        "",
		prometheusExporterEndpoint:            "",
		prometheusRemoteWriteExporterEndpoint: "",
	}
	securityContext := &SecurityContext{
		privileged:      false,
		capabilitiesAdd: make([]string, 0),
	}

	// Create the instance
	return &Instance{
		name:                  name,
		k8sName:               k8sName,
		imageName:             "",
		state:                 StateNone,
		instanceType:          BasicInstance,
		portsTCP:              make([]int, 0),
		portsUDP:              make([]int, 0),
		command:               make([]string, 0),
		args:                  make([]string, 0),
		env:                   make(map[string]string),
		volumes:               make([]*k8s.Volume, 0),
		memoryRequest:         resource.Quantity{},
		memoryLimit:           resource.Quantity{},
		cpuRequest:            resource.Quantity{},
		policyRules:           make([]rbacv1.PolicyRule, 0),
		livenessProbe:         nil,
		readinessProbe:        nil,
		startupProbe:          nil,
		files:                 make([]*k8s.File, 0),
		isSidecar:             false,
		parentInstance:        nil,
		sidecars:              make([]*Instance, 0),
		obsyConfig:            obsyConfig,
		tsharkCollectorConfig: nil,
		securityContext:       securityContext,
		BitTwister:            getBitTwisterDefaultConfig(),
		SystemDependencies:    sysDeps,
	}, nil
}

func (i *Instance) EnableBitTwister() error {
	if i.IsInState(StateStarted) {
		return ErrEnablingBitTwister
	}
	i.BitTwister.enable()
	return nil
}

func (i *Instance) DisableBitTwister() error {
	i.BitTwister.disable()
	return nil
}

// Name returns the name of the instance
func (i *Instance) Name() string {
	return i.name
}

func (i *Instance) K8sName() string {
	return i.k8sName
}

func (i *Instance) SetInstanceType(instanceType InstanceType) {
	i.instanceType = instanceType
}

// SetImage sets the image of the instance.
// When calling in state 'Started', make sure to call AddVolume() before.
// It is only allowed in the 'None' and 'Started' states.
func (i *Instance) SetImage(ctx context.Context, image string) error {
	if !i.IsInState(StateNone, StateStarted) {
		return ErrSettingImageNotAllowed.WithParams(i.state.String())
	}

	if i.state == StateNone {
		// Use the builder to build a new image
		factory, err := container.NewBuilderFactory(image, i.getBuildDir(), i.ImageBuilder)
		if err != nil {
			return ErrCreatingBuilder.Wrap(err)
		}
		i.builderFactory = factory
		i.state = StatePreparing

		return nil
	}

	if i.isSidecar {
		return ErrSettingImageNotAllowedForSidecarsStarted
	}
	return i.setImageWithGracePeriod(ctx, image, nil)
}

// SetGitRepo builds the image from the given git repo, pushes it
// to the registry under the given name and sets the image of the instance.
func (i *Instance) SetGitRepo(ctx context.Context, gitContext builder.GitContext) error {
	if !i.IsInState(StateNone) {
		return ErrSettingGitRepo.WithParams(i.state.String())
	}

	bCtx, err := gitContext.BuildContext()
	if err != nil {
		return ErrGettingBuildContext.Wrap(err)
	}
	imageName, err := builder.DefaultImageName(bCtx)
	if err != nil {
		return ErrGettingImageName.Wrap(err)
	}

	factory, err := container.NewBuilderFactory(imageName, i.getBuildDir(), i.ImageBuilder)
	if err != nil {
		return ErrCreatingBuilder.Wrap(err)
	}
	i.builderFactory = factory
	i.state = StatePreparing

	return i.builderFactory.BuildImageFromGitRepo(ctx, gitContext, imageName)
}

// SetImageInstant sets the image of the instance without a grace period.
// Instant means that the pod is replaced without a grace period of 1 second.
// It is only allowed in the 'Running' state.
func (i *Instance) SetImageInstant(ctx context.Context, image string) error {
	if !i.IsInState(StateStarted) {
		return ErrSettingImageNotAllowedForSidecarsStarted.WithParams(i.state.String())
	}

	if i.isSidecar {
		return ErrSettingImageNotAllowedForSidecars
	}

	gracePeriod := int64(0)
	return i.setImageWithGracePeriod(ctx, image, &gracePeriod)
}

// SetCommand sets the command to run in the instance
// This function can only be called when the instance is in state 'Preparing' or 'Committed'
func (i *Instance) SetCommand(command ...string) error {
	if !i.IsInState(StatePreparing, StateCommitted) {
		return ErrSettingCommand.WithParams(i.state.String())
	}
	i.command = command
	return nil
}

// SetArgs sets the arguments passed to the instance
// This function can only be called in the states 'Preparing' or 'Committed'
func (i *Instance) SetArgs(args ...string) error {
	if !i.IsInState(StatePreparing, StateCommitted) {
		return ErrSettingArgsNotAllowed.WithParams(i.state.String())
	}
	i.args = args
	return nil
}

// AddPortTCP adds a TCP port to the instance
// This function can be called in the states 'Preparing' and 'Committed'
func (i *Instance) AddPortTCP(port int) error {
	if !i.IsInState(StatePreparing, StateCommitted) {
		return ErrAddingPortNotAllowed.WithParams(i.state.String())
	}
	err := validatePort(port)
	if err != nil {
		return err
	}
	if i.isTCPPortRegistered(port) {
		return ErrPortAlreadyRegistered.WithParams(port)
	}
	i.portsTCP = append(i.portsTCP, port)
	i.Logger.Debugf("Added TCP port '%d' to instance '%s'", port, i.name)
	return nil
}

// PortForwardTCP forwards the given port to a random port on the host
// This function can only be called in the state 'Started'
func (i *Instance) PortForwardTCP(ctx context.Context, port int) (int, error) {
	if !i.IsInState(StateStarted) {
		return -1, ErrRandomPortForwardingNotAllowed.WithParams(i.state.String())
	}
	err := validatePort(port)
	if err != nil {
		return 0, err
	}
	if !i.isTCPPortRegistered(port) {
		return -1, ErrPortNotRegistered.WithParams(port)
	}
	// Get a random port on the host
	localPort, err := getFreePortTCP()
	if err != nil {
		return -1, ErrGettingFreePort.WithParams(port)
	}

	// Forward the port
	pod, err := i.K8sClient.GetFirstPodFromReplicaSet(ctx, i.k8sName)
	if err != nil {
		return -1, ErrGettingPodFromReplicaSet.WithParams(i.k8sName).Wrap(err)
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := i.K8sClient.PortForwardPod(ctx, pod.Name, localPort, port)
		if err == nil {
			break
		}
		if attempt == maxRetries {
			return -1, ErrForwardingPort.WithParams(maxRetries)
		}
		i.Logger.Debugf("Forwarding port %d failed, cause: %v, retrying after %v (retry %d/%d)", port, err, retryInterval, attempt, maxRetries)
		time.Sleep(retryInterval)
	}
	return localPort, nil
}

// AddPortUDP adds a UDP port to the instance
// This function can be called in the states 'Preparing' and 'Committed'
func (i *Instance) AddPortUDP(port int) error {
	if !i.IsInState(StatePreparing, StateCommitted) {
		return ErrAddingPortNotAllowed.WithParams(i.state.String())
	}
	err := validatePort(port)
	if err != nil {
		return err
	}
	if i.isUDPPortRegistered(port) {
		return ErrUDPPortAlreadyRegistered.WithParams(port)
	}
	i.portsUDP = append(i.portsUDP, port)
	i.Logger.Debugf("Added UDP port '%d' to instance '%s'", port, i.k8sName)
	return nil
}

// ExecuteCommand executes the given command in the instance
// This function can only be called in the states 'Preparing' and 'Started'
// The context can be used to cancel the command and it is only possible in start state
func (i *Instance) ExecuteCommand(ctx context.Context, command ...string) (string, error) {
	if !i.IsInState(StatePreparing, StateStarted) {
		return "", ErrExecutingCommandNotAllowed.WithParams(i.state.String())
	}

	if i.IsInState(StatePreparing) {
		output, err := i.builderFactory.ExecuteCmdInBuilder(command)
		if err != nil {
			return "", ErrExecutingCommandInInstance.WithParams(command, i.name).Wrap(err)
		}
		return output, nil
	}

	var (
		instanceName  string
		eErr          *Error
		containerName = i.k8sName
	)

	if i.isSidecar {
		instanceName = i.parentInstance.k8sName
		eErr = ErrExecutingCommandInSidecar.WithParams(command, i.k8sName, i.parentInstance.k8sName)
	} else {
		instanceName = i.k8sName
		eErr = ErrExecutingCommandInInstance.WithParams(command, i.k8sName)
	}

	pod, err := i.K8sClient.GetFirstPodFromReplicaSet(ctx, instanceName)
	if err != nil {
		return "", ErrGettingPodFromReplicaSet.WithParams(i.k8sName).Wrap(err)
	}

	commandWithShell := []string{"/bin/sh", "-c", strings.Join(command, " ")}
	output, err := i.K8sClient.RunCommandInPod(ctx, pod.Name, containerName, commandWithShell)
	if err != nil {
		return "", eErr.Wrap(err)
	}
	return output, nil
}

// checkStateForAddingFile checks if the current state allows adding a file
func (i *Instance) checkStateForAddingFile() error {
	if !i.IsInState(StatePreparing, StateCommitted) {
		return ErrAddingFileNotAllowed.WithParams(i.state.String())
	}
	return nil
}

// AddFile adds a file to the instance
// This function can only be called in the state 'Preparing'
func (i *Instance) AddFile(src string, dest string, chown string) error {
	if err := i.checkStateForAddingFile(); err != nil {
		return err
	}

	err := i.validateFileArgs(src, dest, chown)
	if err != nil {
		return err
	}

	// check if src exists (either as file or as folder)
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return ErrSrcDoesNotExist.WithParams(src).Wrap(err)
	}

	// copy file to build dir
	dstPath := filepath.Join(i.getBuildDir(), dest)

	// make sure dir exists
	err = os.MkdirAll(filepath.Dir(dstPath), os.ModePerm)
	if err != nil {
		return ErrCreatingDirectory.Wrap(err)
	}
	// Create destination file making sure the path is writeable.
	dst, err := os.Create(dstPath)
	if err != nil {
		return ErrFailedToCreateDestFile.WithParams(dstPath).Wrap(err)
	}
	defer dst.Close()

	// Open source file for reading.
	srcFile, err := os.Open(src)
	if err != nil {
		return ErrFailedToOpenSrcFile.WithParams(src).Wrap(err)
	}
	defer srcFile.Close()

	// Copy the contents from source file to destination file
	_, err = io.Copy(dst, srcFile)
	if err != nil {
		return ErrFailedToCopyFile.WithParams(src, dstPath).Wrap(err)
	}

	switch i.state {
	case StatePreparing:
		err := i.addFileToBuilder(src, dest, chown)
		if err != nil {
			return err
		}
	case StateCommitted:
		// check if the dest is a sub folder of added volumes and print a warning if not
		if !i.isSubFolderOfVolumes(dest) {
			return ErrFileIsNotSubFolderOfVolumes.WithParams(dest)
		}

		// only allow files, not folders
		srcInfo, err := os.Stat(src)
		if os.IsNotExist(err) || srcInfo.IsDir() {
			return ErrSrcDoesNotExistOrIsDirectory.WithParams(src).Wrap(err)
		}
		file := i.K8sClient.NewFile(dstPath, dest)

		// the user provided a chown string (e.g. "10001:10001") and we only need the group (second part)
		parts := strings.Split(chown, ":")
		if len(parts) != 2 {
			return ErrInvalidFormat
		}

		// second part of array, base of number is 10, and we want a 64-bit integer
		group, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return ErrFailedToConvertToInt64.Wrap(err)
		}

		if i.fsGroup != 0 && i.fsGroup != group {
			return ErrAllFilesMustHaveSameGroup
		} else {
			i.fsGroup = group
		}

		i.files = append(i.files, file)
	}

	i.Logger.Debugf("Added file '%s' to instance '%s'", dest, i.name)
	return nil
}

// AddFolder adds a folder to the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) AddFolder(src string, dest string, chown string) error {
	if !i.IsInState(StatePreparing, StateCommitted) {
		return ErrAddingFolderNotAllowed.WithParams(i.state.String())
	}

	i.validateFileArgs(src, dest, chown)

	// check if src exists (should be a folder)
	srcInfo, err := os.Stat(src)
	if os.IsNotExist(err) || !srcInfo.IsDir() {
		return ErrSrcDoesNotExistOrIsNotDirectory.WithParams(src).Wrap(err)
	}

	// iterate over the files/directories in the src
	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// create the destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(i.getBuildDir(), dest, relPath)

		if info.IsDir() {
			// create directory at destination path
			return os.MkdirAll(dstPath, os.ModePerm)
		}
		// copy file to destination path
		return i.AddFile(path, filepath.Join(dest, relPath), chown)
	})

	if err != nil {
		return ErrCopyingFolderToInstance.WithParams(src, i.name).Wrap(err)
	}

	i.Logger.Debugf("Added folder '%s' to instance '%s'", dest, i.name)
	return nil
}

// AddFileBytes adds a file with the given content to the instance
// This function can only be called in the state 'Preparing'
func (i *Instance) AddFileBytes(bytes []byte, dest string, chown string) error {
	if err := i.checkStateForAddingFile(); err != nil {
		return err
	}

	// create a temporary file
	tmpfile, err := os.CreateTemp("", "temp")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name()) // clean up

	// write bytes to the temporary file
	if _, err := tmpfile.Write(bytes); err != nil {
		return err
	}
	if err := tmpfile.Close(); err != nil {
		return err
	}

	// use AddFile to copy the temp file to the destination
	return i.AddFile(tmpfile.Name(), dest, chown)
}

// SetUser sets the user for the instance
// This function can only be called in the state 'Preparing'
func (i *Instance) SetUser(user string) error {
	if !i.IsInState(StatePreparing) {
		return ErrSettingUserNotAllowed.WithParams(i.state.String())
	}
	err := i.builderFactory.SetUser(user)
	if err != nil {
		return ErrSettingUser.WithParams(user, i.name).Wrap(err)
	}
	i.Logger.Debugf("Set user '%s' for instance '%s'", user, i.name)
	return nil
}

// imageCache maps image hash values to image names
var imageCache = make(map[string]string)

// checkImageHashInCache checks if the given image hash exists in the cache.
func checkImageHashInCache(imageHash string) (imageName string, exists bool) {
	imageName, exists = imageCache[imageHash]
	return imageName, exists
}

// updateImageCacheWithHash adds or updates the image cache with the given hash and image name.
func updateImageCacheWithHash(imageHash, imageName string) {
	imageCache[imageHash] = imageName // Update the cache with the new hash and image name
}

// Commit commits the instance
// This function can only be called in the state 'Preparing'
func (i *Instance) Commit() error {
	if !i.IsInState(StatePreparing) {
		return ErrCommittingNotAllowed.WithParams(i.state.String())
	}
	if i.builderFactory.Changed() {
		// TODO: To speed up the process, the image name could be dependent on the hash of the image
		imageName, err := i.getImageRegistry()
		if err != nil {
			return ErrGettingImageRegistry.Wrap(err)
		}

		// Generate a hash for the current image
		imageHash, err := i.builderFactory.GenerateImageHash()
		if err != nil {
			return ErrGeneratingImageHash.Wrap(err)
		}

		// Check if the generated image hash already exists in the cache, otherwise, we build it.
		cachedImageName, exists := checkImageHashInCache(imageHash)
		if exists {
			i.imageName = cachedImageName
			i.Logger.Debugf("Using cached image for instance '%s'", i.name)
		} else {
			i.Logger.Debugf("Cannot use any cached image for instance '%s'", i.name)
			err = i.builderFactory.PushBuilderImage(imageName)
			if err != nil {
				return ErrPushingImage.WithParams(i.name).Wrap(err)
			}
			updateImageCacheWithHash(imageHash, imageName)
			i.imageName = imageName
			i.Logger.Debugf("Pushed new image for instance '%s'", i.name)
		}
	} else {
		i.imageName = i.builderFactory.ImageNameFrom()
		i.Logger.Debugf("No need to build and push image for instance '%s'", i.name)
	}
	i.state = StateCommitted
	i.Logger.Debugf("Set state of instance '%s' to '%s'", i.name, i.state.String())

	return nil
}

// AddVolume adds a volume to the instance
// The owner of the volume is set to 0, if you want to set a custom owner use AddVolumeWithOwner
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) AddVolume(path string, size resource.Quantity) error {
	// temporary feat, we will remove it once we can add multiple volumes
	if len(i.volumes) > 0 {
		i.Logger.Debugf("Maximum volumes exceeded for instance '%s', volumes: %d", i.name, len(i.volumes))
		return ErrMaximumVolumesExceeded.WithParams(i.name)
	}
	i.AddVolumeWithOwner(path, size, 0)
	return nil
}

// AddVolumeWithOwner adds a volume to the instance with the given owner
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) AddVolumeWithOwner(path string, size resource.Quantity, owner int64) error {
	if !i.IsInState(StatePreparing, StateCommitted) {
		return ErrAddingVolumeNotAllowed.WithParams(i.state.String())
	}
	// temporary feat, we will remove it once we can add multiple volumes
	if len(i.volumes) > 0 {
		i.Logger.Debugf("Maximum volumes exceeded for instance '%s', volumes: %d", i.name, len(i.volumes))
		return ErrMaximumVolumesExceeded.WithParams(i.name)
	}
	volume := i.K8sClient.NewVolume(path, size, owner)
	i.volumes = append(i.volumes, volume)
	logrus.Debugf("Added volume '%s' with size '%s' and owner '%d' to instance '%s'", path, size.String(), owner, i.name)
	return nil
}

// SetMemory sets the memory of the instance
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) SetMemory(request, limit resource.Quantity) error {
	if !i.IsInState(StatePreparing, StateCommitted) {
		return ErrSettingMemoryNotAllowed.WithParams(i.state.String())
	}
	i.memoryRequest = request
	i.memoryLimit = limit
	logrus.Debugf("Set memory to '%s' and limit to '%s' in instance '%s'", request.String(), limit.String(), i.name)
	return nil
}

// SetCPU sets the CPU of the instance
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) SetCPU(request resource.Quantity) error {
	if !i.IsInState(StatePreparing, StateCommitted) {
		return ErrSettingCPUNotAllowed.WithParams(i.state.String())
	}
	i.cpuRequest = request
	logrus.Debugf("Set cpu to '%s' in instance '%s'", request.String(), i.name)
	return nil
}

// SetEnvironmentVariable sets the given environment variable in the instance
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) SetEnvironmentVariable(key, value string) error {
	if !i.IsInState(StatePreparing, StateCommitted) {
		return ErrSettingEnvNotAllowed.WithParams(i.state.String())
	}
	if i.state == StatePreparing {
		err := i.builderFactory.SetEnvVar(key, value)
		if err != nil {
			return err
		}
	} else if i.state == StateCommitted {
		i.env[key] = value
	}
	i.Logger.Debugf("Set environment variable '%s' in instance '%s'", key, i.name)
	return nil
}

// GetIP returns the IP of the instance
// This function can only be called in the states 'Preparing' and 'Started'
func (i *Instance) GetIP(ctx context.Context) (string, error) {
	// Check if i.kubernetesService already has the IP
	if i.kubernetesService != nil && i.kubernetesService.Spec.ClusterIP != "" {
		return i.kubernetesService.Spec.ClusterIP, nil
	}
	// If not, proceed with the existing logic to deploy the service and get the IP
	svc, err := i.K8sClient.GetService(ctx, i.k8sName)
	if err != nil || svc == nil {
		// Service does not exist, so we need to deploy it
		err := i.deployService(ctx, i.portsTCP, i.portsUDP)
		if err != nil {
			return "", ErrDeployingServiceForInstance.WithParams(i.k8sName).Wrap(err)
		}
		svc, err = i.K8sClient.GetService(ctx, i.k8sName)
		if err != nil {
			return "", ErrGettingServiceForInstance.WithParams(i.k8sName).Wrap(err)
		}
	}

	ip := svc.Spec.ClusterIP
	if ip == "" {
		return "", ErrGettingServiceIP.WithParams(i.k8sName)
	}

	// Update i.kubernetesService for future reference
	i.kubernetesService = svc

	return ip, nil
}

// GetFileBytes returns the content of the given file
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) GetFileBytes(ctx context.Context, file string) ([]byte, error) {
	if !i.IsInState(StatePreparing, StateCommitted, StateStarted) {
		return nil, ErrGettingFileNotAllowed.WithParams(i.state.String())
	}

	if i.state != StateStarted {
		bytes, err := i.builderFactory.ReadFileFromBuilder(file)
		if err != nil {
			return nil, ErrGettingFile.WithParams(file, i.name).Wrap(err)
		}
		return bytes, nil
	}

	rc, err := i.ReadFileFromRunningInstance(ctx, file)
	if err != nil {
		return nil, ErrReadingFile.WithParams(file, i.name).Wrap(err)
	}

	defer rc.Close()
	return io.ReadAll(rc)
}

func (i *Instance) ReadFileFromRunningInstance(ctx context.Context, filePath string) (io.ReadCloser, error) {
	if !i.IsInState(StateStarted) {
		return nil, ErrReadingFileNotAllowed.WithParams(i.state.String())
	}

	// Not the best solution, we need to find a better one.
	// Tested with a 110MB+ file and it worked.
	fileContent, err := i.ExecuteCommand(ctx, "cat", filePath)
	if err != nil {
		return nil, ErrReadingFileFromInstance.WithParams(filePath, i.name).Wrap(err)
	}
	return io.NopCloser(strings.NewReader(fileContent)), nil
}

// AddPolicyRule adds a policy rule to the instance
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) AddPolicyRule(rule rbacv1.PolicyRule) error {
	if !i.IsInState(StatePreparing, StateCommitted) {
		return ErrAddingPolicyRuleNotAllowed.WithParams(i.state.String())
	}
	i.policyRules = append(i.policyRules, rule)
	return nil
}

// checkStateForProbe checks if the current state is allowed for setting a probe
func (i *Instance) checkStateForProbe() error {
	if !i.IsInState(StatePreparing, StateCommitted) {
		return ErrSettingProbeNotAllowed.WithParams(i.state.String())
	}
	return nil
}

// SetLivenessProbe sets the liveness probe of the instance
// A live probe is a probe that is used to determine if the instance is still alive, and should be restarted if not
// See usage documentation: https://pkg.go.dev/i.K8sCli.io/api/core/v1@v0.27.3#Probe
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) SetLivenessProbe(livenessProbe *v1.Probe) error {
	if err := i.checkStateForProbe(); err != nil {
		return err
	}
	i.livenessProbe = livenessProbe
	i.Logger.Debugf("Set liveness probe to '%s' in instance '%s'", livenessProbe, i.name)
	return nil
}

// SetReadinessProbe sets the readiness probe of the instance
// A readiness probe is a probe that is used to determine if the instance is ready to receive traffic
// See usage documentation: https://pkg.go.dev/i.K8sCli.io/api/core/v1@v0.27.3#Probe
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) SetReadinessProbe(readinessProbe *v1.Probe) error {
	if err := i.checkStateForProbe(); err != nil {
		return err
	}
	i.readinessProbe = readinessProbe
	i.Logger.Debugf("Set readiness probe to '%s' in instance '%s'", readinessProbe, i.name)
	return nil
}

// SetStartupProbe sets the startup probe of the instance
// A startup probe is a probe that is used to determine if the instance is ready to receive traffic after a startup
// See usage documentation: https://pkg.go.dev/i.K8sCli.io/api/core/v1@v0.27.3#Probe
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) SetStartupProbe(startupProbe *v1.Probe) error {
	if err := i.checkStateForProbe(); err != nil {
		return err
	}
	i.startupProbe = startupProbe
	i.Logger.Debugf("Set startup probe to '%s' in instance '%s'", startupProbe, i.name)
	return nil
}

// AddSidecar adds a sidecar to the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) AddSidecar(sidecar *Instance) error {

	if !i.IsInState(StatePreparing, StateCommitted) {
		return ErrAddingSidecarNotAllowed.WithParams(i.state.String())
	}
	if sidecar == nil {
		return ErrSidecarIsNil
	}
	if sidecar == i {
		return ErrSidecarCannotBeSameInstance
	}
	if sidecar.state != StateCommitted {
		return ErrSidecarNotCommitted.WithParams(sidecar.name)
	}
	if i.isSidecar {
		return ErrSidecarCannotHaveSidecar.WithParams(i.name)
	}
	if sidecar.isSidecar {
		return ErrSidecarAlreadySidecar.WithParams(sidecar.name)
	}

	i.sidecars = append(i.sidecars, sidecar)
	sidecar.isSidecar = true
	sidecar.parentInstance = i
	i.Logger.Debugf("Added sidecar '%s' to instance '%s'", sidecar.name, i.name)
	return nil
}

// SetOtelCollectorVersion sets the OpenTelemetry collector version for the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) SetOtelCollectorVersion(version string) error {
	if err := i.validateStateForObsy("OpenTelemetry collector version"); err != nil {
		return err
	}
	i.obsyConfig.otelCollectorVersion = version
	i.Logger.Debugf("Set OpenTelemetry collector version '%s' for instance '%s'", version, i.name)
	return nil
}

// SetOtelEndpoint sets the OpenTelemetry endpoint for the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) SetOtelEndpoint(port int) error {
	if err := i.validateStateForObsy("OpenTelemetry endpoint"); err != nil {
		return err
	}
	i.obsyConfig.otlpPort = port
	i.Logger.Debugf("Set OpenTelemetry endpoint '%d' for instance '%s'", port, i.name)
	return nil
}

// SetPrometheusEndpoint sets the Prometheus endpoint for the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) SetPrometheusEndpoint(port int, jobName, scapeInterval string) error {
	if err := i.validateStateForObsy("Prometheus endpoint"); err != nil {
		return err
	}
	i.obsyConfig.prometheusEndpointPort = port
	i.obsyConfig.prometheusEndpointJobName = jobName
	i.obsyConfig.prometheusEndpointScrapeInterval = scapeInterval
	i.Logger.Debugf("Set Prometheus endpoint '%d' for instance '%s'", port, i.name)
	return nil
}

// SetJaegerEndpoint sets the Jaeger endpoint for the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) SetJaegerEndpoint(grpcPort, thriftCompactPort, thriftHttpPort int) error {
	if err := i.validateStateForObsy("Jaeger endpoint"); err != nil {
		return err
	}
	i.obsyConfig.jaegerGrpcPort = grpcPort
	i.obsyConfig.jaegerThriftCompactPort = thriftCompactPort
	i.obsyConfig.jaegerThriftHttpPort = thriftHttpPort
	i.Logger.Debugf("Set Jaeger endpoints '%d', '%d' and '%d' for instance '%s'", grpcPort, thriftCompactPort, thriftHttpPort, i.name)
	return nil
}

// SetOtlpExporter sets the OTLP exporter for the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) SetOtlpExporter(endpoint, username, password string) error {
	if err := i.validateStateForObsy("OTLP exporter"); err != nil {
		return err
	}
	i.obsyConfig.otlpEndpoint = endpoint
	i.obsyConfig.otlpUsername = username
	i.obsyConfig.otlpPassword = password
	i.Logger.Debugf("Set OTLP exporter '%s' for instance '%s'", endpoint, i.name)
	return nil
}

// SetJaegerExporter sets the Jaeger exporter for the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) SetJaegerExporter(endpoint string) error {
	if err := i.validateStateForObsy("Jaeger exporter"); err != nil {
		return err
	}
	i.obsyConfig.jaegerEndpoint = endpoint
	i.Logger.Debugf("Set Jaeger exporter '%s' for instance '%s'", endpoint, i.name)
	return nil
}

// SetPrometheusExporter sets the Prometheus exporter for the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) SetPrometheusExporter(endpoint string) error {
	if err := i.validateStateForObsy("Prometheus exporter"); err != nil {
		return err
	}
	i.obsyConfig.prometheusExporterEndpoint = endpoint
	i.Logger.Debugf("Set Prometheus exporter '%s' for instance '%s'", endpoint, i.name)
	return nil
}

// SetPrometheusRemoteWriteExporter sets the Prometheus remote write exporter for the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) SetPrometheusRemoteWriteExporter(endpoint string) error {
	if err := i.validateStateForObsy("Prometheus remote write exporter"); err != nil {
		return err
	}
	i.obsyConfig.prometheusRemoteWriteExporterEndpoint = endpoint
	i.Logger.Debugf("Set Prometheus remote write exporter '%s' for instance '%s'", endpoint, i.name)
	return nil
}

// TsharkCollectorEnabled returns true if the tshark collector is enabled
func (i *Instance) TsharkCollectorEnabled() bool {
	return i.tsharkCollectorConfig != nil
}

// EnableTsharkCollector enables the tshark collector for the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) EnableTsharkCollector(conf TsharkCollectorConfig) error {
	if err := i.validateStateForObsy(tsharkCollectorName); err != nil {
		return err
	}
	if i.TsharkCollectorEnabled() {
		return ErrTsharkCollectorAlreadyEnabled
	}

	if err := validateTsharkCollectorConfig(conf); err != nil {
		return err
	}

	i.tsharkCollectorConfig = &conf
	i.Logger.Debugf("Enabled Tshark collector for instance '%s'", i.name)
	return nil
}

// validateTsharkCollectorConfig checks the configuration fields for proper formatting
func validateTsharkCollectorConfig(conf TsharkCollectorConfig) error {
	// Regex patterns for validation
	awsKeyPattern, err := regexp.Compile(`^[A-Za-z0-9]{1,20}$`)
	if err != nil {
		return ErrRegexpCompile.WithParams("awsKeyPattern")
	}
	awsSecretPattern, err := regexp.Compile(`^[A-Za-z0-9/+=]{1,40}$`)
	if err != nil {
		return ErrRegexpCompile.WithParams("awsSecretPattern")
	}

	if !awsKeyPattern.MatchString(conf.S3AccessKey) {
		return ErrTsharkCollectorInvalidS3AccessKey.WithParams(conf.S3AccessKey)
	}
	if !awsSecretPattern.MatchString(conf.S3SecretKey) {
		return ErrTsharkCollectorInvalidS3SecretKey.WithParams(conf.S3SecretKey)
	}
	if conf.S3Region == "" || conf.S3Bucket == "" {
		return ErrTsharkCollectorS3RegionOrBucketEmpty.WithParams(conf.S3Region, conf.S3Bucket)
	}

	return nil
}

// SetPrivileged sets the privileged status for the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) SetPrivileged(privileged bool) error {
	if !i.IsInState(StatePreparing, StateCommitted) {
		return ErrSettingPrivilegedNotAllowed.WithParams(i.state.String())
	}
	i.securityContext.privileged = privileged
	i.Logger.Debugf("Set privileged to '%t' for instance '%s'", privileged, i.name)
	return nil
}

// AddCapability adds a capability to the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) AddCapability(capability string) error {
	if !i.IsInState(StatePreparing, StateCommitted) {
		return ErrAddingCapabilityNotAllowed.WithParams(i.state.String())
	}
	i.securityContext.capabilitiesAdd = append(i.securityContext.capabilitiesAdd, capability)
	i.Logger.Debugf("Added capability '%s' to instance '%s'", capability, i.name)
	return nil
}

// AddCapabilities adds multiple capabilities to the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) AddCapabilities(capabilities []string) error {
	if !i.IsInState(StatePreparing, StateCommitted) {
		return ErrAddingCapabilitiesNotAllowed.WithParams(i.state.String())
	}
	for _, capability := range capabilities {
		i.securityContext.capabilitiesAdd = append(i.securityContext.capabilitiesAdd, capability)
		i.Logger.Debugf("Added capability '%s' to instance '%s'", capability, i.name)
	}
	return nil
}

// StartAsync starts the instance without waiting for it to be ready
// This function can only be called in the state 'Committed' or 'Stopped'
// This function will replace StartWithoutWait
func (i *Instance) StartAsync(ctx context.Context) error {
	if err := i.StartWithoutWait(ctx); err != nil {
		return err
	}
	return nil
}

// StartWithoutWait starts the instance without waiting for it to be ready
// This function can only be called in the state 'Committed' or 'Stopped'
func (i *Instance) StartWithoutWait(ctx context.Context) error {
	if !i.IsInState(StateCommitted, StateStopped) {
		return ErrStartingNotAllowed.WithParams(i.state.String())
	}
	if err := applyFunctionToInstances(i.sidecars, func(sidecar Instance) error {
		if !sidecar.IsInState(StateCommitted, StateStopped) {
			return ErrStartingNotAllowedForSidecar.WithParams(sidecar.name, sidecar.state.String())
		}
		return nil
	}); err != nil {
		return err
	}
	if i.isSidecar {
		return ErrStartingSidecarNotAllowed
	}

	if i.state == StateCommitted {
		// deploy otel collector if observability is enabled
		if i.isObservabilityEnabled() {
			if err := i.addOtelCollectorSidecar(ctx); err != nil {
				return ErrAddingOtelCollectorSidecar.WithParams(i.k8sName).Wrap(err)
			}
		}

		// deploy tshark collector if enabled
		if i.TsharkCollectorEnabled() {
			if err := i.addTsharkCollectorSidecar(ctx); err != nil {
				return ErrAddingTsharkCollectorSidecar.WithParams(i.k8sName).Wrap(err)
			}
		}

		if i.BitTwister.Enabled() {
			if err := i.addBitTwisterSidecar(ctx); err != nil {
				return ErrAddingNetworkSidecar.WithParams(i.k8sName).Wrap(err)
			}
		}

		if err := i.deployResources(ctx); err != nil {
			return ErrDeployingResourcesForInstance.WithParams(i.k8sName).Wrap(err)
		}
		if err := applyFunctionToInstances(i.sidecars, func(sidecar *Instance) error {
			return sidecar.deployResources(ctx)
		}); err != nil {
			return ErrDeployingResourcesForSidecars.WithParams(i.k8sName).Wrap(err)
		}
	}

	err := i.deployPod(ctx)
	if err != nil {
		return ErrDeployingPodForInstance.WithParams(i.k8sName).Wrap(err)
	}
	i.state = StateStarted
	setStateForSidecars(i.sidecars, StateStarted)
	i.Logger.Debugf("Set state of instance '%s' to '%s'", i.k8sName, i.state.String())

	return nil
}

// Start starts the instance and waits for it to be ready
// This function can only be called in the state 'Committed' and 'Stopped'
func (i *Instance) Start(ctx context.Context) error {
	if err := i.StartWithoutWait(ctx); err != nil {
		return err
	}

	err := i.WaitInstanceIsRunning(ctx)
	if err != nil {
		return ErrWaitingForInstanceRunning.WithParams(i.k8sName).Wrap(err)
	}

	return nil
}

// IsRunning returns true if the instance is running
// This function can only be called in the state 'Started'
func (i *Instance) IsRunning(ctx context.Context) (bool, error) {
	if !i.IsInState(StateStarted, StateStopped) {
		return false, ErrCheckingIfInstanceRunningNotAllowed.WithParams(i.state.String())
	}

	return i.K8sClient.IsReplicaSetRunning(ctx, i.k8sName)
}

// WaitInstanceIsRunning waits until the instance is running
// This function can only be called in the state 'Started'
func (i *Instance) WaitInstanceIsRunning(ctx context.Context) error {
	if !i.IsInState(StateStarted) {
		return ErrWaitingForInstanceNotAllowed.WithParams(i.state.String())
	}
	tick := time.NewTicker(1 * time.Second)

	for {
		running, err := i.IsRunning(ctx)
		if err != nil {
			return ErrCheckingIfInstanceRunning.WithParams(i.k8sName).Wrap(err)
		}
		if running {
			return nil
		}

		select {
		case <-ctx.Done():
			return ErrWaitingForInstanceTimeout.WithParams(i.k8sName).Wrap(ctx.Err())
		case <-tick.C:
			continue
		}
	}
}

// DisableNetwork disables the network of the instance
// This does not apply to executor instances
// This function can only be called in the state 'Started'
func (i *Instance) DisableNetwork(ctx context.Context) error {
	if !i.IsInState(StateStarted) {
		return ErrDisablingNetworkNotAllowed.WithParams(i.state.String())
	}
	executorSelectorMap := map[string]string{
		"knuu.sh/type": ExecutorInstance.String(),
	}

	err := i.K8sClient.CreateNetworkPolicy(ctx, i.k8sName, i.getLabels(), executorSelectorMap, executorSelectorMap)
	if err != nil {
		return ErrDisablingNetwork.WithParams(i.k8sName).Wrap(err)
	}
	return nil
}

// SetBandwidthLimit sets the bandwidth limit of the instance
// bandwidth limit in bps (e.g. 1000 for 1Kbps)
// Currently, only one of bandwidth, jitter, latency or packet loss can be set
// This function can only be called in the state 'Commited'
func (i *Instance) SetBandwidthLimit(limit int64) error {
	if !i.IsInState(StateStarted) {
		return ErrSettingBandwidthLimitNotAllowed.WithParams(i.state.String())
	}
	if !i.BitTwister.Enabled() {
		return ErrSettingBandwidthLimitNotAllowedBitTwister
	}

	// We first need to stop it, otherwise we get an error
	if err := i.BitTwister.Client().BandwidthStop(); err != nil {
		if !sdk.IsErrorServiceNotInitialized(err) &&
			!sdk.IsErrorServiceNotReady(err) &&
			!sdk.IsErrorServiceNotStarted(err) {
			return ErrStoppingBandwidthLimit.WithParams(i.k8sName).Wrap(err)
		}
	}

	err := i.BitTwister.Client().BandwidthStart(sdk.BandwidthStartRequest{
		NetworkInterfaceName: i.BitTwister.NetworkInterface(),
		Limit:                limit,
	})
	if err != nil {
		return ErrSettingBandwidthLimit.WithParams(i.k8sName).Wrap(err)
	}

	i.Logger.Debugf("Set bandwidth limit to '%d' in instance '%s'", limit, i.name)
	return nil
}

// SetLatency sets the latency of the instance
// latency in ms (e.g. 1000 for 1s)
// jitter in ms (e.g. 1000 for 1s)
// Currently, only one of bandwidth, jitter, latency or packet loss can be set
// This function can only be called in the state 'Commited'
func (i *Instance) SetLatencyAndJitter(latency, jitter int64) error {
	if !i.IsInState(StateStarted) {
		return ErrSettingLatencyJitterNotAllowed.WithParams(i.state.String())
	}
	if !i.BitTwister.Enabled() {
		return ErrSettingLatencyJitterNotAllowedBitTwister
	}

	// We first need to stop it, otherwise we get an error
	if err := i.BitTwister.Client().LatencyStop(); err != nil {
		if !sdk.IsErrorServiceNotInitialized(err) &&
			!sdk.IsErrorServiceNotReady(err) &&
			!sdk.IsErrorServiceNotStarted(err) {
			return ErrStoppingLatencyJitter.WithParams(i.k8sName).Wrap(err)
		}
	}

	err := i.BitTwister.Client().LatencyStart(sdk.LatencyStartRequest{
		NetworkInterfaceName: i.BitTwister.NetworkInterface(),
		Latency:              latency,
		Jitter:               jitter,
	})
	if err != nil {
		return ErrSettingLatencyJitter.WithParams(i.k8sName).Wrap(err)
	}

	i.Logger.Debugf("Set latency to '%d' and jitter to '%d' in instance '%s'", latency, jitter, i.name)
	return nil
}

// SetPacketLoss sets the packet loss of the instance
// packet loss in percent (e.g. 10 for 10%)
// Currently, only one of bandwidth, jitter, latency or packet loss can be set
// This function can only be called in the state 'Commited'
func (i *Instance) SetPacketLoss(packetLoss int32) error {
	if !i.IsInState(StateStarted) {
		return ErrSettingPacketLossNotAllowed.WithParams(i.state.String())
	}
	if !i.BitTwister.Enabled() {
		return ErrSettingPacketLossNotAllowedBitTwister
	}

	// We first need to stop it, otherwise we get an error
	if err := i.BitTwister.Client().PacketlossStop(); err != nil {
		if !sdk.IsErrorServiceNotInitialized(err) &&
			!sdk.IsErrorServiceNotReady(err) &&
			!sdk.IsErrorServiceNotStarted(err) {
			return ErrStoppingPacketLoss.WithParams(i.k8sName).Wrap(err)
		}
	}

	err := i.BitTwister.Client().PacketlossStart(sdk.PacketLossStartRequest{
		NetworkInterfaceName: i.BitTwister.NetworkInterface(),
		PacketLossRate:       packetLoss,
	})
	if err != nil {
		return ErrSettingPacketLoss.WithParams(i.k8sName).Wrap(err)
	}

	i.Logger.Debugf("Set packet loss to '%d' in instance '%s'", packetLoss, i.name)
	return nil
}

// EnableNetwork enables the network of the instance
// This function can only be called in the state 'Started'
func (i *Instance) EnableNetwork(ctx context.Context) error {
	if !i.IsInState(StateStarted) {
		return ErrEnablingNetworkNotAllowed.WithParams(i.state.String())
	}

	err := i.K8sClient.DeleteNetworkPolicy(ctx, i.k8sName)
	if err != nil {
		return ErrEnablingNetwork.WithParams(i.k8sName).Wrap(err)
	}
	return nil
}

// NetworkIsDisabled returns true if the network of the instance is disabled
// This function can only be called in the state 'Started'
func (i *Instance) NetworkIsDisabled(ctx context.Context) (bool, error) {
	if !i.IsInState(StateStarted) {
		return false, ErrCheckingIfNetworkDisabledNotAllowed.WithParams(i.state.String())
	}

	return i.K8sClient.NetworkPolicyExists(ctx, i.k8sName), nil
}

// WaitInstanceIsStopped waits until the instance is not running anymore
// This function can only be called in the state 'Stopped'
func (i *Instance) WaitInstanceIsStopped(ctx context.Context) error {
	if !i.IsInState(StateStopped) {
		return ErrWaitingForInstanceStoppedNotAllowed.WithParams(i.state.String())
	}
	for {
		running, err := i.IsRunning(ctx)
		if !running {
			break
		}
		if err != nil {
			return ErrCheckingIfInstanceStopped.WithParams(i.k8sName).Wrap(err)
		}
	}

	return nil
}

// Stop stops the instance
// CAUTION: In order to keep data of the instance, you need to use AddVolume() before.
// This function can only be called in the state 'Started'
func (i *Instance) Stop(ctx context.Context) error {
	if !i.IsInState(StateStarted) {
		return ErrStoppingNotAllowed.WithParams(i.state.String())

	}

	if err := i.destroyPod(ctx); err != nil {
		return ErrDestroyingPod.WithParams(i.k8sName).Wrap(err)
	}
	i.state = StateStopped
	setStateForSidecars(i.sidecars, StateStopped)
	i.Logger.Debugf("Set state of instance '%s' to '%s'", i.k8sName, i.state.String())

	return nil
}

// Clone creates a clone of the instance
// This function can only be called in the state 'Committed'
// When cloning an instance that is a sidecar, the clone will be not a sidecar
// When cloning an instance with sidecars, the sidecars will be cloned as well
func (i *Instance) Clone() (*Instance, error) {
	if !i.IsInState(StateCommitted) {
		return nil, ErrCloningNotAllowed.WithParams(i.state.String())
	}

	newK8sName, err := names.NewRandomK8(i.name)
	if err != nil {
		return nil, ErrGeneratingK8sName.WithParams(i.name).Wrap(err)
	}
	// Create a new instance with the same attributes as the original instance
	ins := i.cloneWithSuffix("")
	ins.k8sName = newK8sName
	return ins, nil
}

// CloneWithName creates a clone of the instance with a given name
// This function can only be called in the state 'Committed'
// When cloning an instance that is a sidecar, the clone will be not a sidecar
// When cloning an instance with sidecars, the sidecars will be cloned as well
func (i *Instance) CloneWithName(name string) (*Instance, error) {
	if !i.IsInState(StateCommitted) {
		return nil, ErrCloningNotAllowedForSidecar.WithParams(i.state.String())
	}

	newK8sName, err := names.NewRandomK8(name)
	if err != nil {
		return nil, ErrGeneratingK8sNameForSidecar.WithParams(name).Wrap(err)
	}
	// Create a new instance with the same attributes as the original instance
	ins := i.cloneWithSuffix("")
	ins.name = name
	ins.k8sName = newK8sName
	return ins, nil
}

// CreateCustomResource creates a custom resource for the instance
// The names and namespace are set and overridden by knuu
func (i *Instance) CreateCustomResource(ctx context.Context, gvr *schema.GroupVersionResource, obj *map[string]interface{}) error {
	crdExists, err := i.CustomResourceDefinitionExists(ctx, gvr)
	if err != nil {
		return err
	}
	if !crdExists {
		return ErrCustomResourceDefinitionDoesNotExist.WithParams(gvr.Resource)
	}

	return i.K8sClient.CreateCustomResource(ctx, i.k8sName, gvr, obj)
}

// CustomResourceDefinitionExists checks if the custom resource definition exists
func (i *Instance) CustomResourceDefinitionExists(ctx context.Context, gvr *schema.GroupVersionResource) (bool, error) {
	return i.K8sClient.CustomResourceDefinitionExists(ctx, gvr)
}
