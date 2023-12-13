package knuu

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/celestiaorg/bittwister/sdk"
	"github.com/celestiaorg/knuu/pkg/container"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/sirupsen/logrus"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// ObsyConfig represents the configuration for the obsy sidecar
type ObsyConfig struct {
	// otelCollectorVersion is the version of the otel collector to use
	otelCollectorVersion string

	// prometheusPort is the port on which the prometheus server will be exposed
	prometheusPort int
	// prometheusJobName is the name of the prometheus job
	prometheusJobName string
	// prometheusScrapeInterval is the scrape interval for the prometheus job
	prometheusScrapeInterval string

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
	name                  string
	imageName             string
	k8sName               string
	state                 InstanceState
	instanceType          InstanceType
	kubernetesService     *v1.Service
	builderFactory        *container.BuilderFactory
	kubernetesStatefulSet *appv1.StatefulSet
	portsTCP              []int
	portsUDP              []int
	command               []string
	args                  []string
	env                   map[string]string
	volumes               []*k8s.Volume
	memoryRequest         string
	memoryLimit           string
	cpuRequest            string
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
	securityContext       *SecurityContext
	BitTwister            *btConfig
}

// NewInstance creates a new instance of the Instance struct
func NewInstance(name string) (*Instance, error) {
	// Generate a UUID for this instance

	k8sName, err := generateK8sName(name)
	if err != nil {
		return nil, fmt.Errorf("error generating k8s name for instance '%s': %w", name, err)
	}
	obsyConfig := &ObsyConfig{
		otelCollectorVersion:     "0.83.0",
		otlpPort:                 0,
		prometheusPort:           0,
		prometheusJobName:        "",
		prometheusScrapeInterval: "",
		jaegerGrpcPort:           0,
		jaegerThriftCompactPort:  0,
		jaegerThriftHttpPort:     0,
		otlpEndpoint:             "",
		otlpUsername:             "",
		otlpPassword:             "",
		jaegerEndpoint:           "",
	}
	securityContext := &SecurityContext{
		privileged:      false,
		capabilitiesAdd: make([]string, 0),
	}

	// Create the instance
	return &Instance{
		name:            name,
		k8sName:         k8sName,
		imageName:       "",
		state:           None,
		instanceType:    BasicInstance,
		portsTCP:        make([]int, 0),
		portsUDP:        make([]int, 0),
		command:         make([]string, 0),
		args:            make([]string, 0),
		env:             make(map[string]string),
		volumes:         make([]*k8s.Volume, 0),
		memoryRequest:   "",
		memoryLimit:     "",
		cpuRequest:      "",
		policyRules:     make([]rbacv1.PolicyRule, 0),
		livenessProbe:   nil,
		readinessProbe:  nil,
		startupProbe:    nil,
		files:           make([]*k8s.File, 0),
		isSidecar:       false,
		parentInstance:  nil,
		sidecars:        make([]*Instance, 0),
		obsyConfig:      obsyConfig,
		securityContext: securityContext,
		BitTwister:      getBitTwisterDefaultConfig(),
	}, nil
}

func (i *Instance) EnableBitTwister() error {
	if i.IsInState(Started) {
		return fmt.Errorf("enabling BitTwister is not allowed in state 'Started'")
	}
	i.BitTwister.enable()
	return nil
}

func (i *Instance) DisableBitTwister() error {
	// if !i.IsInState(Preparing) {
	// 	return fmt.Errorf("disabling BitTwister is only allowed in state 'Preparing'. Current state is '%s'", i.state.String())
	// }
	i.BitTwister.disable()
	return nil
}

// SetImage sets the image of the instance.
// When calling in state 'Started', make sure to call AddVolume() before.
// It is only allowed in the 'None' and 'Started' states.
func (i *Instance) SetImage(image string) error {
	// Check if setting the image is allowed in the current state
	if !i.IsInState(None, Started) {
		return fmt.Errorf("setting image is only allowed in state 'None' and 'Started'. Current state is '%s'", i.state.String())
	}

	// Handle each state accordingly
	switch i.state {
	case None:
		// Use the builder to build a new image
		factory, err := container.NewBuilderFactory(image, i.getBuildDir())
		//builder, storage, err := container.NewBuilder(context, image)
		if err != nil {
			return fmt.Errorf("error creating builder: %s", err.Error())
		}
		i.builderFactory = factory
		i.state = Preparing
	case Started:

		if i.isSidecar {
			return fmt.Errorf("setting image is not allowed for sidecars when in state 'Started'")
		}

		if err := i.setImageWithGracePeriod(image, nil); err != nil {
			return err
		}
	}

	return nil
}

// SetImageInstant sets the image of the instance without a grace period.
// Instant means that the pod is replaced without a grace period of 1 second.
// It is only allowed in the 'Running' state.
func (i *Instance) SetImageInstant(image string) error {
	// Check if setting the image is allowed in the current state
	if !i.IsInState(Started) {
		return fmt.Errorf("setting image is only allowed in state 'Started'. Current state is '%s'", i.state.String())
	}

	if i.isSidecar {
		return fmt.Errorf("setting image is not allowed for sidecars")
	}

	gracePeriod := int64(0)

	if err := i.setImageWithGracePeriod(image, &gracePeriod); err != nil {
		return err
	}

	return nil
}

// SetCommand sets the command to run in the instance
// This function can only be called when the instance is in state 'Preparing' or 'Committed'
func (i *Instance) SetCommand(command ...string) error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("setting command is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	i.command = command
	return nil
}

// SetArgs sets the arguments passed to the instance
// This function can only be called in the states 'Preparing' or 'Committed'
func (i *Instance) SetArgs(args ...string) error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("setting args is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	i.args = args
	return nil
}

// AddPortTCP adds a TCP port to the instance
// This function can be called in the states 'Preparing' and 'Committed'
func (i *Instance) AddPortTCP(port int) error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("adding port is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	err := validatePort(port)
	if err != nil {
		return err
	}
	if i.isTCPPortRegistered(port) {
		return fmt.Errorf("TCP port '%d' is already in registered", port)
	}
	i.portsTCP = append(i.portsTCP, port)
	logrus.Debugf("Added TCP port '%d' to instance '%s'", port, i.name)
	return nil
}

// PortForwardTCP forwards the given port to a random port on the host
// This function can only be called in the state 'Started'
func (i *Instance) PortForwardTCP(port int) (int, error) {
	if !i.IsInState(Started) {
		return -1, fmt.Errorf("random port forwarding is only allowed in state 'Started'. Current state is '%s'", i.state.String())
	}
	err := validatePort(port)
	if err != nil {
		return 0, err
	}
	if !i.isTCPPortRegistered(port) {
		return -1, fmt.Errorf("TCP port '%d' is not registered", port)
	}
	// Get a random port on the host
	localPort, err := getFreePortTCP()
	if err != nil {
		return -1, fmt.Errorf("error getting free port: %v", err)
	}
	// Forward the port
	pod, err := k8s.GetFirstPodFromStatefulSet(k8s.Namespace(), i.k8sName)
	if err != nil {
		return -1, fmt.Errorf("error getting pod from statefulset '%s': %v", i.k8sName, err)
	}
	// We need to retry here because the port forwarding might fail as getFreePortTCP() might not free the port fast enough
	retries := 5
	wait := 5 * time.Second
	for r := 0; r < retries; r++ {
		err = k8s.PortForwardPod(k8s.Namespace(), pod.Name, localPort, port)
		if err == nil {
			break
		}
		if retries == r+1 {
			return -1, fmt.Errorf("error forwarding port after %d retries: %v", retries, err)
		}
		logrus.Debugf("Forwaring port %d failed, cause: %v, retrying after %v (retry %d/%d)", port, err, wait, r+1, retries)
		time.Sleep(wait)
	}
	return localPort, nil
}

// AddPortUDP adds a UDP port to the instance
// This function can be called in the states 'Preparing' and 'Committed'
func (i *Instance) AddPortUDP(port int) error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("adding port is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	err := validatePort(port)
	if err != nil {
		return err
	}
	if i.isUDPPortRegistered(port) {
		return fmt.Errorf("UDP port '%d' is already in registered", port)
	}
	i.portsUDP = append(i.portsUDP, port)
	logrus.Debugf("Added UDP port '%d' to instance '%s'", port, i.k8sName)
	return nil
}

// ExecuteCommand executes the given command in the instance
// This function can only be called in the states 'Preparing' and 'Started'
func (i *Instance) ExecuteCommand(command ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), k8s.Timeout())
	defer cancel()

	return i.ExecuteCommandWithContext(ctx, command...)
}

// ExecuteCommandWithContext executes the given command in the instance
// This function can only be called in the states 'Preparing' and 'Started'
// The context can be used to cancel the command and it is only possible in start state
func (i *Instance) ExecuteCommandWithContext(ctx context.Context, command ...string) (string, error) {
	if !i.IsInState(Preparing, Started) {
		return "", fmt.Errorf("executing command is only allowed in state 'Preparing' or 'Started'. Current state is '%s'", i.state.String())
	}

	if i.IsInState(Preparing) {
		output, err := i.builderFactory.ExecuteCmdInBuilder(command)
		if err != nil {
			return "", fmt.Errorf("error executing command '%s' in instance '%s': %v", command, i.name, err)
		}
		return output, nil
	}

	var (
		instanceName  string
		errMsg        error
		containerName = i.k8sName
	)

	if i.isSidecar {
		instanceName = i.parentInstance.k8sName
		errMsg = fmt.Errorf("error executing command '%s' in sidecar '%s' of instance '%s'", command, i.k8sName, i.parentInstance.k8sName)
	} else {
		instanceName = i.k8sName
		errMsg = fmt.Errorf("error executing command '%s' in instance '%s'", command, i.k8sName)
	}

	pod, err := k8s.GetFirstPodFromStatefulSet(k8s.Namespace(), instanceName)
	if err != nil {
		return "", fmt.Errorf("error getting pod from statefulset '%s': %v", i.k8sName, err)
	}

	commandWithShell := []string{"/bin/sh", "-c", strings.Join(command, " ")}
	output, err := k8s.RunCommandInPod(ctx, k8s.Namespace(), pod.Name, containerName, commandWithShell)
	if err != nil {
		return "", fmt.Errorf("%v: %v", errMsg, err)
	}
	return output, nil
}

// checkStateForAddingFile checks if the current state allows adding a file
func (i *Instance) checkStateForAddingFile() error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("adding file is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
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
		return fmt.Errorf("src '%s' does not exist", src)
	}

	// copy file to build dir
	dstPath := filepath.Join(i.getBuildDir(), dest)

	// make sure dir exists
	err = os.MkdirAll(filepath.Dir(dstPath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}
	// Create destination file making sure the path is writeable.
	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file '%s': %w", dstPath, err)
	}
	defer dst.Close()

	// Open source file for reading.
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file '%s': %w", src, err)
	}
	defer srcFile.Close()

	// Copy the contents from source file to destination file
	_, err = io.Copy(dst, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy from source '%s' to destination '%s': %w", src, dstPath, err)
	}

	switch i.state {
	case Preparing:
		err := i.addFileToBuilder(src, dest, chown)
		if err != nil {
			return err
		}
	case Committed:
		// only allow files, not folders
		srcInfo, err := os.Stat(src)
		if os.IsNotExist(err) || srcInfo.IsDir() {
			return fmt.Errorf("src '%s' does not exist or is a directory", src)
		}
		file := k8s.NewFile(dstPath, dest)

		// the user provided a chown string (e.g. "10001:10001") and we only need the group (second part)
		parts := strings.Split(chown, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid format")
		}

		// second part of array, base of number is 10, and we want a 64-bit integer
		group, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return fmt.Errorf("failed to convert to int64: %s", err)
		}

		if i.fsGroup != 0 && i.fsGroup != group {
			return fmt.Errorf("all files must have the same group")
		} else {
			i.fsGroup = group
		}

		i.files = append(i.files, file)
	}

	logrus.Debugf("Added file '%s' to instance '%s'", dest, i.name)
	return nil
}

// AddFolder adds a folder to the instance
// This function can only be called in the state 'Preparing'
func (i *Instance) AddFolder(src string, dest string, chown string) error {
	if !i.IsInState(Preparing) {
		return fmt.Errorf("adding folder is only allowed in state 'Preparing'. Current state is '%s'", i.state.String())
	}

	i.validateFileArgs(src, dest, chown)

	// check if src exists (should be a folder)
	srcInfo, err := os.Stat(src)
	if os.IsNotExist(err) || !srcInfo.IsDir() {
		return fmt.Errorf("src '%s' does not exist or is not a directory", src)
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
		} else {
			// copy file to destination path
			return i.AddFile(path, filepath.Join(dest, relPath), chown)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error copying folder '%s' to instance '%s': %w", src, i.name, err)
	}

	logrus.Debugf("Added folder '%s' to instance '%s'", dest, i.name)
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
	if !i.IsInState(Preparing) {
		return fmt.Errorf("setting user is only allowed in state 'Preparing'. Current state is '%s'", i.state.String())
	}
	err := i.builderFactory.SetUser(user)
	if err != nil {
		return fmt.Errorf("error setting user '%s' for instance '%s': %w", user, i.name, err)
	}
	logrus.Debugf("Set user '%s' for instance '%s'", user, i.name)
	return nil
}

// Commit commits the instance
// This function can only be called in the state 'Preparing'
func (i *Instance) Commit() error {
	if !i.IsInState(Preparing) {
		return fmt.Errorf("committing is only allowed in state 'Preparing'. Current state is '%s'", i.state.String())
	}
	if i.builderFactory.Changed() {
		// TODO: To speed up the process, the image name could be dependent on the hash of the image
		imageName, err := i.getImageRegistry()
		if err != nil {
			return fmt.Errorf("error getting image registry: %w", err)
		}
		err = i.builderFactory.PushBuilderImage(imageName)
		if err != nil {
			return fmt.Errorf("error pushing image for instance '%s': %w", i.name, err)
		}
		i.imageName = imageName
		logrus.Debugf("Pushed image for instance '%s'", i.name)
	} else {
		i.imageName = i.builderFactory.ImageNameFrom()
		logrus.Debugf("No need to build and push image for instance '%s'", i.name)
	}
	i.state = Committed
	logrus.Debugf("Set state of instance '%s' to '%s'", i.name, i.state.String())

	return nil
}

// AddVolume adds a volume to the instance
// The owner of the volume is set to 0, if you want to set a custom owner use AddVolumeWithOwner
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) AddVolume(path, size string) error {
	i.AddVolumeWithOwner(path, size, 0)
	return nil
}

// AddVolumeWithOwner adds a volume to the instance with the given owner
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) AddVolumeWithOwner(path, size string, owner int64) error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("adding volume is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	volume := k8s.NewVolume(path, size, owner)
	i.volumes = append(i.volumes, volume)
	logrus.Debugf("Added volume '%s' with size '%s' and owner '%d' to instance '%s'", path, size, owner, i.name)
	return nil
}

// SetMemory sets the memory of the instance
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) SetMemory(request, limit string) error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("setting memory is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	i.memoryRequest = request
	i.memoryLimit = limit
	logrus.Debugf("Set memory to '%s' and limit to '%s' in instance '%s'", request, limit, i.name)
	return nil
}

// SetCPU sets the CPU of the instance
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) SetCPU(request string) error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("setting cpu is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	i.cpuRequest = request
	logrus.Debugf("Set cpu to '%s' in instance '%s'", request, i.name)
	return nil
}

// SetEnvironmentVariable sets the given environment variable in the instance
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) SetEnvironmentVariable(key, value string) error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("setting environment variable is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	if i.state == Preparing {
		err := i.builderFactory.SetEnvVar(key, value)
		if err != nil {
			return err
		}
	} else if i.state == Committed {
		i.env[key] = value
	}
	logrus.Debugf("Set environment variable '%s' to '%s' in instance '%s'", key, value, i.name)
	return nil
}

// GetIP returns the IP of the instance
// This function can only be called in the states 'Preparing' and 'Started'
func (i *Instance) GetIP() (string, error) {
	svc, _ := k8s.GetService(k8s.Namespace(), i.k8sName)
	if svc == nil {
		// Service does not exist, so we need to deploy it
		err := i.deployService()
		if err != nil {
			return "", fmt.Errorf("error deploying service '%s': %w", i.k8sName, err)
		}
	}

	ip, err := k8s.GetServiceIP(k8s.Namespace(), i.k8sName)
	if err != nil {
		return "", fmt.Errorf("error getting IP of service '%s': %w", i.k8sName, err)
	}

	return ip, nil
}

// GetFileBytes returns the content of the given file
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) GetFileBytes(file string) ([]byte, error) {
	if !i.IsInState(Preparing, Committed) {
		return nil, fmt.Errorf("getting file is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}

	bytes, err := i.builderFactory.ReadFileFromBuilder(file)
	if err != nil {
		return nil, fmt.Errorf("error getting file '%s' from instance '%s': %w", file, i.name, err)
	}
	return bytes, nil
}

// AddPolicyRule adds a policy rule to the instance
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) AddPolicyRule(rule rbacv1.PolicyRule) error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("adding policy rule is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	i.policyRules = append(i.policyRules, rule)
	return nil
}

// checkStateForProbe checks if the current state is allowed for setting a probe
func (i *Instance) checkStateForProbe() error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("setting probe is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	return nil
}

// SetLivenessProbe sets the liveness probe of the instance
// A live probe is a probe that is used to determine if the instance is still alive, and should be restarted if not
// See usage documentation: https://pkg.go.dev/k8s.io/api/core/v1@v0.27.3#Probe
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) SetLivenessProbe(livenessProbe *v1.Probe) error {
	if err := i.checkStateForProbe(); err != nil {
		return err
	}
	i.livenessProbe = livenessProbe
	logrus.Debugf("Set liveness probe to '%s' in instance '%s'", livenessProbe, i.name)
	return nil
}

// SetReadinessProbe sets the readiness probe of the instance
// A readiness probe is a probe that is used to determine if the instance is ready to receive traffic
// See usage documentation: https://pkg.go.dev/k8s.io/api/core/v1@v0.27.3#Probe
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) SetReadinessProbe(readinessProbe *v1.Probe) error {
	if err := i.checkStateForProbe(); err != nil {
		return err
	}
	i.readinessProbe = readinessProbe
	logrus.Debugf("Set readiness probe to '%s' in instance '%s'", readinessProbe, i.name)
	return nil
}

// SetStartupProbe sets the startup probe of the instance
// A startup probe is a probe that is used to determine if the instance is ready to receive traffic after a startup
// See usage documentation: https://pkg.go.dev/k8s.io/api/core/v1@v0.27.3#Probe
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) SetStartupProbe(startupProbe *v1.Probe) error {
	if err := i.checkStateForProbe(); err != nil {
		return err
	}
	i.startupProbe = startupProbe
	logrus.Debugf("Set startup probe to '%s' in instance '%s'", startupProbe, i.name)
	return nil
}

// AddSidecar adds a sidecar to the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) AddSidecar(sidecar *Instance) error {

	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("adding sidecar is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	if sidecar == nil {
		return fmt.Errorf("sidecar is nil")
	}
	if sidecar == i {
		return fmt.Errorf("sidecar cannot be the same instance")
	}
	if sidecar.state != Committed {
		return fmt.Errorf("sidecar '%s' is not in state 'Committed'", sidecar.name)
	}
	if i.isSidecar {
		return fmt.Errorf("sidecar '%s' cannot have a sidecar", i.name)
	}
	if sidecar.isSidecar {
		return fmt.Errorf("sidecar '%s' is already a sidecar", sidecar.name)
	}

	i.sidecars = append(i.sidecars, sidecar)
	sidecar.isSidecar = true
	sidecar.parentInstance = i
	logrus.Debugf("Added sidecar '%s' to instance '%s'", sidecar.name, i.name)
	return nil
}

// SetOtelCollectorVersion sets the OpenTelemetry collector version for the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) SetOtelCollectorVersion(version string) error {
	if err := i.validateStateForObsy("OpenTelemetry collector version"); err != nil {
		return err
	}
	i.obsyConfig.otelCollectorVersion = version
	logrus.Debugf("Set OpenTelemetry collector version '%s' for instance '%s'", version, i.name)
	return nil
}

// SetOtelEndpoint sets the OpenTelemetry endpoint for the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) SetOtelEndpoint(port int) error {
	if err := i.validateStateForObsy("OpenTelemetry endpoint"); err != nil {
		return err
	}
	i.obsyConfig.otlpPort = port
	logrus.Debugf("Set OpenTelemetry endpoint '%d' for instance '%s'", port, i.name)
	return nil
}

// SetPrometheusEndpoint sets the Prometheus endpoint for the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) SetPrometheusEndpoint(port int, jobName, scapeInterval string) error {
	if err := i.validateStateForObsy("Prometheus endpoint"); err != nil {
		return err
	}
	i.obsyConfig.prometheusPort = port
	i.obsyConfig.prometheusJobName = jobName
	i.obsyConfig.prometheusScrapeInterval = scapeInterval
	logrus.Debugf("Set Prometheus endpoint '%d' for instance '%s'", port, i.name)
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
	logrus.Debugf("Set Jaeger endpoints '%d', '%d' and '%d' for instance '%s'", grpcPort, thriftCompactPort, thriftHttpPort, i.name)
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
	logrus.Debugf("Set OTLP exporter '%s' for instance '%s'", endpoint, i.name)
	return nil
}

// SetJaegerExporter sets the Jaeger exporter for the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) SetJaegerExporter(endpoint string) error {
	if err := i.validateStateForObsy("Jaeger exporter"); err != nil {
		return err
	}
	i.obsyConfig.jaegerEndpoint = endpoint
	logrus.Debugf("Set Jaeger exporter '%s' for instance '%s'", endpoint, i.name)
	return nil
}

// SetPrivileged sets the privileged status for the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) SetPrivileged(privileged bool) error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("setting privileged is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	i.securityContext.privileged = privileged
	logrus.Debugf("Set privileged to '%t' for instance '%s'", privileged, i.name)
	return nil
}

// AddCapability adds a capability to the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) AddCapability(capability string) error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("adding capability is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	i.securityContext.capabilitiesAdd = append(i.securityContext.capabilitiesAdd, capability)
	logrus.Debugf("Added capability '%s' to instance '%s'", capability, i.name)
	return nil
}

// AddCapabilities adds multiple capabilities to the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (i *Instance) AddCapabilities(capabilities []string) error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("adding capabilities is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	for _, capability := range capabilities {
		i.securityContext.capabilitiesAdd = append(i.securityContext.capabilitiesAdd, capability)
		logrus.Debugf("Added capability '%s' to instance '%s'", capability, i.name)
	}
	return nil
}

// StartWithoutWait starts the instance without waiting for it to be ready
// This function can only be called in the state 'Committed' or 'Stopped'
func (i *Instance) StartWithoutWait() error {
	if !i.IsInState(Committed, Stopped) {
		return fmt.Errorf("starting is only allowed in state 'Committed' or 'Stopped'. Current state is '%s'", i.state.String())
	}
	if err := applyFunctionToInstances(i.sidecars, func(sidecar Instance) error {
		if !sidecar.IsInState(Committed, Stopped) {
			return fmt.Errorf("starting is only allowed in state 'Committed' or 'Stopped'. Current state of sidecar '%s' is '%s'", sidecar.name, sidecar.state.String())
		}
		return nil
	}); err != nil {
		return err
	}
	if i.isSidecar {
		return fmt.Errorf("starting a sidecar is not allowed")
	}

	if i.state == Committed {
		// deploy otel collector if observability is enabled
		if i.isObservabilityEnabled() {
			if err := i.addOtelCollectorSidecar(); err != nil {
				return fmt.Errorf("error adding OpenTelemetry collector sidecar for instance '%s': %w", i.k8sName, err)
			}
		}

		if i.BitTwister.Enabled() || i.isObservabilityEnabled() {
			if err := i.addNetworkConfigSidecar(); err != nil {
				return fmt.Errorf("error adding network sidecar for instance '%s': %w", i.k8sName, err)
			}
		}

		if err := i.deployResources(); err != nil {
			return fmt.Errorf("error deploying resources for instance '%s': %w", i.k8sName, err)
		}
		if err := applyFunctionToInstances(i.sidecars, func(sidecar Instance) error {
			return sidecar.deployResources()
		}); err != nil {
			return fmt.Errorf("error deploying resources for sidecars of instance '%s': %w", i.k8sName, err)
		}
	}

	err := i.deployPod()
	if err != nil {
		return fmt.Errorf("error deploying pod for instance '%s': %w", i.k8sName, err)
	}
	i.state = Started
	setStateForSidecars(i.sidecars, Started)
	logrus.Debugf("Set state of instance '%s' to '%s'", i.k8sName, i.state.String())

	return nil
}

// Start starts the instance and waits for it to be ready
// This function can only be called in the state 'Committed' and 'Stopped'
func (i *Instance) Start() error {
	if err := i.StartWithoutWait(); err != nil {
		return err
	}

	err := i.WaitInstanceIsRunning()
	if err != nil {
		return fmt.Errorf("error waiting for instance '%s' to be running: %w", i.k8sName, err)
	}

	return nil
}

// IsRunning returns true if the instance is running
// This function can only be called in the state 'Started'
func (i *Instance) IsRunning() (bool, error) {
	if !i.IsInState(Started, Stopped) {
		return false, fmt.Errorf("checking if instance is running is only allowed in state 'Started'. Current state is '%s'", i.state.String())
	}
	return k8s.IsStatefulSetRunning(k8s.Namespace(), i.k8sName)
}

// WaitInstanceIsRunning waits until the instance is running
// This function can only be called in the state 'Started'
func (i *Instance) WaitInstanceIsRunning() error {
	if !i.IsInState(Started) {
		return fmt.Errorf("waiting for instance is only allowed in state 'Started'. Current state is '%s'", i.state.String())
	}
	timeout := time.After(1 * time.Minute)
	tick := time.Tick(1 * time.Second)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout while waiting for instance '%s' to be running", i.k8sName)
		case <-tick:
			running, err := i.IsRunning()
			if err != nil {
				return fmt.Errorf("error checking if instance '%s' is running: %w", i.k8sName, err)
			}
			if running {
				return nil
			}
		}
	}
}

// DisableNetwork disables the network of the instance
// This does not apply to executor instances
// This function can only be called in the state 'Started'
func (i *Instance) DisableNetwork() error {
	if !i.IsInState(Started) {
		return fmt.Errorf("disabling network is only allowed in state 'Started'. Current state is '%s'", i.state.String())
	}
	executorSelectorMap := map[string]string{
		"knuu.sh/type": ExecutorInstance.String(),
	}
	err := k8s.CreateNetworkPolicy(k8s.Namespace(), i.k8sName, i.getLabels(), executorSelectorMap, executorSelectorMap)
	if err != nil {
		return fmt.Errorf("error disabling network for instance '%s': %w", i.k8sName, err)
	}
	return nil
}

// SetBandwidthLimit sets the bandwidth limit of the instance
// bandwidth limit in bps (e.g. 1000 for 1Kbps)
// Currently, only one of bandwidth, jitter, latency or packet loss can be set
// This function can only be called in the state 'Commited'
func (i *Instance) SetBandwidthLimit(limit int64) error {
	if !i.IsInState(Started) {
		return fmt.Errorf("setting bandwidth limit is only allowed in state 'Started'. Current state is '%s'", i.state.String())
	}
	if !i.BitTwister.Enabled() {
		return fmt.Errorf("setting bandwidth limit is only allowed if BitTwister is enabled")
	}

	// We first need to stop it, otherwise we get an error
	if err := i.BitTwister.Client().BandwidthStop(); err != nil {
		if !sdk.IsErrorServiceNotInitialized(err) &&
			!sdk.IsErrorServiceNotReady(err) &&
			!sdk.IsErrorServiceNotStarted(err) {
			return fmt.Errorf("error stopping bandwidth limit for instance '%s': %w", i.k8sName, err)
		}
	}

	err := i.BitTwister.Client().BandwidthStart(sdk.BandwidthStartRequest{
		NetworkInterfaceName: i.BitTwister.NetworkInterface(),
		Limit:                limit,
	})
	if err != nil {
		return fmt.Errorf("error setting bandwidth limit for instance '%s': %w", i.k8sName, err)
	}

	logrus.Debugf("Set bandwidth limit to '%d' in instance '%s'", limit, i.name)
	return nil
}

// SetLatency sets the latency of the instance
// latency in ms (e.g. 1000 for 1s)
// jitter in ms (e.g. 1000 for 1s)
// Currently, only one of bandwidth, jitter, latency or packet loss can be set
// This function can only be called in the state 'Commited'
func (i *Instance) SetLatencyAndJitter(latency, jitter int64) error {
	if !i.IsInState(Started) {
		return fmt.Errorf("setting latency/jitter is only allowed in state 'Started'. Current state is '%s'", i.state.String())
	}
	if !i.BitTwister.Enabled() {
		return fmt.Errorf("setting latency/jitter is only allowed if BitTwister is enabled")
	}

	// We first need to stop it, otherwise we get an error
	if err := i.BitTwister.Client().LatencyStop(); err != nil {
		if !sdk.IsErrorServiceNotInitialized(err) &&
			!sdk.IsErrorServiceNotReady(err) &&
			!sdk.IsErrorServiceNotStarted(err) {
			return fmt.Errorf("error stopping latency/jitter for instance '%s': %w", i.k8sName, err)
		}
	}

	err := i.BitTwister.Client().LatencyStart(sdk.LatencyStartRequest{
		NetworkInterfaceName: i.BitTwister.NetworkInterface(),
		Latency:              latency,
		Jitter:               jitter,
	})
	if err != nil {
		return fmt.Errorf("error setting latency/jitter for instance '%s': %w", i.k8sName, err)
	}

	logrus.Debugf("Set latency to '%d' and jitter to '%d' in instance '%s'", latency, jitter, i.name)
	return nil
}

// SetPacketLoss sets the packet loss of the instance
// packet loss in percent (e.g. 10 for 10%)
// Currently, only one of bandwidth, jitter, latency or packet loss can be set
// This function can only be called in the state 'Commited'
func (i *Instance) SetPacketLoss(packetLoss int32) error {
	if !i.IsInState(Started) {
		return fmt.Errorf("setting packetloss is only allowed in state 'Started'. Current state is '%s'", i.state.String())
	}
	if !i.BitTwister.Enabled() {
		return fmt.Errorf("setting packetloss is only allowed if BitTwister is enabled")
	}

	// We first need to stop it, otherwise we get an error
	if err := i.BitTwister.Client().PacketlossStop(); err != nil {
		if !sdk.IsErrorServiceNotInitialized(err) &&
			!sdk.IsErrorServiceNotReady(err) &&
			!sdk.IsErrorServiceNotStarted(err) {
			return fmt.Errorf("error stopping packetloss for instance '%s': %w", i.k8sName, err)
		}
	}

	err := i.BitTwister.Client().PacketlossStart(sdk.PacketLossStartRequest{
		NetworkInterfaceName: i.BitTwister.NetworkInterface(),
		PacketLossRate:       packetLoss,
	})
	if err != nil {
		return fmt.Errorf("error setting packetloss for instance '%s': %w", i.k8sName, err)
	}

	logrus.Debugf("Set packet loss to '%d' in instance '%s'", packetLoss, i.name)
	return nil
}

// EnableNetwork enables the network of the instance
// This function can only be called in the state 'Started'
func (i *Instance) EnableNetwork() error {
	if !i.IsInState(Started) {
		return fmt.Errorf("enabling network is only allowed in state 'Started'. Current state is '%s'", i.state.String())
	}
	err := k8s.DeleteNetworkPolicy(k8s.Namespace(), i.k8sName)
	if err != nil {
		return fmt.Errorf("error enabling network for instance '%s': %w", i.k8sName, err)
	}
	return nil
}

// NetworkIsDisabled returns true if the network of the instance is disabled
// This function can only be called in the state 'Started'
func (i *Instance) NetworkIsDisabled() (bool, error) {
	if !i.IsInState(Started) {
		return false, fmt.Errorf("checking if network is disabled is only allowed in state 'Started'. Current state is '%s'", i.state.String())
	}
	return k8s.NetworkPolicyExists(k8s.Namespace(), i.k8sName), nil
}

// WaitInstanceIsStopped waits until the instance is not running anymore
// This function can only be called in the state 'Stopped'
func (i *Instance) WaitInstanceIsStopped() error {
	if !i.IsInState(Stopped) {
		return fmt.Errorf("waiting for instance is only allowed in state 'Stopped'. Current state is '%s'", i.state.String())
	}
	for {
		running, err := i.IsRunning()
		if !running {
			break
		}
		if err != nil {
			return fmt.Errorf("error checking if instance '%s' is running: %w", i.k8sName, err)
		}
	}

	return nil
}

// Stop stops the instance
// CAUTION: In order to keep data of the instance, you need to use AddVolume() before.
// This function can only be called in the state 'Started'
func (i *Instance) Stop() error {
	if !i.IsInState(Started) {
		return fmt.Errorf("stopping is only allowed in state 'Started'. Current state is '%s'", i.state.String())
	}
	err := i.destroyPod()
	if err != nil {
		return fmt.Errorf("error destroying pod for instance '%s': %w", i.k8sName, err)
	}
	i.state = Stopped
	setStateForSidecars(i.sidecars, Stopped)
	logrus.Debugf("Set state of instance '%s' to '%s'", i.k8sName, i.state.String())

	return nil
}

// Destroy destroys the instance
// This function can only be called in the state 'Started' or 'Destroyed'
func (i *Instance) Destroy() error {
	if !i.IsInState(Started, Stopped, Destroyed) {
		return fmt.Errorf("destroying is only allowed in state 'Started' or 'Destroyed'. Current state is '%s'", i.state.String())
	}
	if i.state == Destroyed {
		return nil
	}
	err := i.destroyPod()
	if err != nil {
		return fmt.Errorf("error destroying pod for instance '%s': %w", i.k8sName, err)
	}
	if err := i.destroyResources(); err != nil {
		return fmt.Errorf("error destroying resources for instance '%s': %w", i.k8sName, err)
	}

	if err := applyFunctionToInstances(i.sidecars, func(sidecar Instance) error {
		logrus.Debugf("Destroying sidecar resources from '%s'", sidecar.k8sName)
		return sidecar.destroyResources()
	}); err != nil {
		return fmt.Errorf("error destroying resources for sidecars of instance '%s': %w", i.k8sName, err)
	}

	i.state = Destroyed
	setStateForSidecars(i.sidecars, Destroyed)
	logrus.Debugf("Set state of instance '%s' to '%s'", i.k8sName, i.state.String())

	return nil
}

// Clone creates a clone of the instance
// This function can only be called in the state 'Committed'
// When cloning an instance that is a sidecar, the clone will be not a sidecar
// When cloning an instance with sidecars, the sidecars will be cloned as well
func (i *Instance) Clone() (*Instance, error) {
	if !i.IsInState(Committed) {
		return nil, fmt.Errorf("cloning is only allowed in state 'Committed'. Current state is '%s'", i.state.String())
	}

	newK8sName, err := generateK8sName(i.name)
	if err != nil {
		return nil, fmt.Errorf("error generating k8s name for instance '%s': %w", i.name, err)
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
	if !i.IsInState(Committed) {
		return nil, fmt.Errorf("cloning is only allowed in state 'Committed'. Current state is '%s'", i.state.String())
	}

	newK8sName, err := generateK8sName(name)
	if err != nil {
		return nil, fmt.Errorf("error generating k8s name for instance '%s': %w", name, err)
	}
	// Create a new instance with the same attributes as the original instance
	ins := i.cloneWithSuffix("")
	ins.name = name
	ins.k8sName = newK8sName
	return ins, nil
}
