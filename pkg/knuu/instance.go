// Package knuu provides the core functionality of knuu.
package knuu

import (
	"context"
	"fmt"
	"os"

	"github.com/celestiaorg/knuu/pkg/container"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/containers/buildah"
	"github.com/containers/storage"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// State represents the state of the instance
type State int

// Possible states of the instance
const (
	None State = iota
	Preparing
	Committed
	Started
	Destroyed
)

// String returns the string representation of the state
func (s State) String() string {
	if s < 0 || s > 4 {
		return "Unknown"
	}
	return [...]string{"None", "Preparing", "Committed", "Started", "Destroyed"}[s]
}

// Instance represents a instance
type Instance struct {
	uuid              uuid.UUID
	name              string
	k8sName           string
	state             State
	kubernetesService *v1.Service
	imageBuilder      *buildah.Builder
	buildStore        storage.Store
	buildContext      context.Context
	kubernetesPod     *v1.Pod
	portsTCP          []int
	portsUDP          []int
	files             []string
	command           []string
	args              []string
	env               map[string]string
	volumes           map[string]string
}

// NewInstance creates a new instance of the Instance struct
func NewInstance(name string) (*Instance, error) {
	// Generate a UUID for this instance
	uuid, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("error generating UUID for instance '%s': %w", name, err)
	}
	k8sName := fmt.Sprintf("%s-%s", name, uuid.String()[:8])
	// Create the instance
	return &Instance{
		uuid:     uuid,
		name:     name,
		k8sName:  k8sName,
		state:    None,
		portsTCP: make([]int, 0),
		portsUDP: make([]int, 0),
		files:    make([]string, 0),
		command:  make([]string, 0),
		args:     make([]string, 0),
		env:      make(map[string]string),
		volumes:  make(map[string]string),
	}, nil
}

// IsInState checks if the instance is in one of the provided states
func (i *Instance) IsInState(states ...State) bool {
	for _, s := range states {
		if i.state == s {
			return true
		}
	}
	return false
}

// SetImage sets the image of the instance.
// It is only allowed in the 'None' and 'Started' states.
func (i *Instance) SetImage(image string) error {
	// Check if setting the image is allowed in the current state
	if !i.IsInState(None, Started) {
		return fmt.Errorf("Setting image is only allowed in state 'None' and 'Started'. Current state is '%s'", i.state.String())
	}

	var err error

	// Handle each state accordingly
	switch i.state {
	case None:
		// Create a new build context
		context, _ := context.WithCancel(context.Background())

		i.buildContext = context

		// Use the builder to build a new image
		builder, storage, err := container.NewBuilder(context, image)
		if err != nil {
			return fmt.Errorf("Error getting builder: %s", err.Error())
		}
		i.imageBuilder = builder
		i.buildStore = storage
		i.state = Preparing
	case Started:

		// Generate the pod configuration
		podConfig := k8s.PodConfig{
			Namespace: k8s.Namespace,
			Name:      i.k8sName,
			Labels:    i.kubernetesPod.Labels,
			Image:     image,
			Command:   i.command,
			Args:      i.args,
			Env:       i.env,
			Volumes:   i.volumes,
		}
		// Replace the pod with a new one, using the given image
		_, err = k8s.ReplacePod(podConfig)
		if err != nil {
			return fmt.Errorf("Error replacing pod: %s", err.Error())
		}
		i.WaitInstanceIsRunning()
	}

	return nil
}

// SetCommand sets the command to run in the instance
// This function can only be called when the instance is in state 'Preparing' or 'Committed'
func (i *Instance) SetCommand(command []string) error {
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

// getTempImageRegistry returns the name of the temporary image registry
func (i *Instance) getTempImageRegistry() string {
	return fmt.Sprintf("ttl.sh/%s:1h", i.uuid.String())
}

// AddPortTCP adds a TCP port to the instance
// This function can be called in the states 'Preparing' and 'Committed'
func (i *Instance) AddPortTCP(port int) error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("adding port is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	validatePort(port)
	if i.isTCPPortRegistered(port) {
		return fmt.Errorf("TCP port '%d' is already in registered", port)
	}
	i.portsTCP = append(i.portsTCP, port)
	logrus.Debugf("Added TCP port '%d' to instance '%s'", port, i.name)
	return nil
}

// isTCPPortRegistered returns true if the given port is registered
// with the instance, and false otherwise
func (i *Instance) isTCPPortRegistered(port int) bool {
	for _, p := range i.portsTCP {
		if p == port {
			return true
		}
	}
	return false
}

// AddPortUDP adds a UDP port to the instance
// This function can be called in the states 'Preparing' and 'Committed'
func (i *Instance) AddPortUDP(port int) error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("adding port is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	validatePort(port)
	if i.isUDPPortRegistered(port) {
		return fmt.Errorf("UDP port '%d' is already in registered", port)
	}
	i.portsUDP = append(i.portsUDP, port)
	logrus.Debugf("Added UDP port '%d' to instance '%s'", port, i.k8sName)
	return nil
}

// isUDPPortRegistered returns true if the given port is registered
// with the instance, and false otherwise
func (i *Instance) isUDPPortRegistered(port int) bool {
	for _, p := range i.portsUDP {
		if p == port {
			return true
		}
	}
	return false
}

// ExecuteCommand executes the given command in the instance
// This function can only be called in the states 'Preparing' and 'Started'
func (i *Instance) ExecuteCommand(command ...string) (string, error) {
	if !i.IsInState(Preparing, Started) {
		return "", fmt.Errorf("executing command is only allowed in state 'Preparing' or 'Started'. Current state is '%s'", i.state.String())
	}
	if i.IsInState(Preparing) {
		output, err := container.ExecuteCmdInBuilder(i.imageBuilder, command)
		if err != nil {
			return "", fmt.Errorf("error executing command '%s' in instance '%s': %v", command, i.name, err)
		}
		return output, nil
	} else if i.IsInState(Started) {
		output, err := k8s.RunCommandInPod(k8s.Namespace, i.k8sName, i.k8sName, command)
		if err != nil {
			return "", fmt.Errorf("error executing command '%s' in started instance '%s': %v", command, i.k8sName, err)
		}
		return output, nil
	} else {
		return "", fmt.Errorf("cannot execute command '%s' in instance '%s' in state '%s'", command, i.k8sName, i.state.String())
	}

	return "", nil
}

// AddFileBytes adds a file with the given content to the instance
// This function can only be called in the state 'Preparing'
func (i *Instance) AddFileBytes(bytes []byte, dest string, chown string) error {
	if !i.IsInState(Preparing) {
		return fmt.Errorf("adding file is only allowed in state 'Preparing'. Current state is '%s'", i.state.String())
	}

	tmpFile, err := os.CreateTemp("", "temp-file-")
	if err != nil {
		return fmt.Errorf("error creating temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(bytes)
	if err != nil {
		return fmt.Errorf("error writing content to temporary file: %w", err)
	}
	tmpFile.Close()

	i.AddFile(tmpFile.Name(), dest, chown)

	return nil
}

// AddFile adds a file to the instance
// This function can only be called in the state 'Preparing'
func (i *Instance) AddFile(src string, dest string, chown string) error {
	if !i.IsInState(Preparing) {
		return fmt.Errorf("adding file is only allowed in state 'Preparing'. Current state is '%s'", i.state.String())
	}

	i.files = append(i.files, dest)
	err := container.AddFileToBuilder(i.imageBuilder, src, dest, chown)
	if err != nil {
		return fmt.Errorf("error adding file '%s' to instance '%s': %w", dest, i.name, err)
	}
	logrus.Debugf("Added file '%s' to instance '%s'", dest, i.name)
	return nil
}

// GetFileBytes returns the content of the given file
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) GetFileBytes(file string) ([]byte, error) {
	if !i.IsInState(Preparing, Committed) {
		return nil, fmt.Errorf("getting file is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}

	bytes, err := container.ReadFileFromBuilder(i.imageBuilder, file)
	if err != nil {
		return nil, fmt.Errorf("error getting file '%s' from instance '%s': %w", file, i.name, err)
	}
	return bytes, nil
}

// SetEnvironmentVariable sets the given environment variable in the instance
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) SetEnvironmentVariable(key string, value string) error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("setting environment variable is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	if i.state == Preparing {
		container.SetEnvVar(i.imageBuilder, key, value)
	} else if i.state == Committed {
		i.env[key] = value
	}
	logrus.Debugf("Set environment variable '%s' to '%s' in instance '%s'", key, value, i.name)
	return nil
}

// AddVolume adds a volume to the instance
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) AddVolume(name string, size string) error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("adding volume is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	i.volumes[name] = size
	logrus.Debugf("Added volume '%s' with size '%s' to instance '%s'", name, size, i.name)
	return nil
}

// GetIP returns the IP of the instance
// This function can only be called in the states 'Preparing' and 'Started'
func (i *Instance) GetIP() (string, error) {
	svc, _ := k8s.GetService(k8s.Namespace, i.k8sName)
	if svc == nil {
		// Service does not exist, so we need to deploy it
		err := i.deployService()
		if err != nil {
			return "", fmt.Errorf("error deploying service '%s': %w", i.k8sName, err)
		}
	}

	ip, err := k8s.GetServiceIP(k8s.Namespace, i.k8sName)
	if err != nil {
		return "", fmt.Errorf("error getting IP of service '%s': %w", i.k8sName, err)
	}

	return ip, nil
}

// deployService deploys the service for the instance
func (i *Instance) deployService() error {
	svc, _ := k8s.GetService(k8s.Namespace, i.k8sName)
	if svc != nil {
		// Service already exists, so we patch it
		err := i.patchService()
		if err != nil {
			return fmt.Errorf("error patching service '%s': %w", i.k8sName, err)
		}
	}

	labels := i.getLabels()
	selectorMap := i.getLabels()
	service, err := k8s.DeployService(k8s.Namespace, i.k8sName, labels, selectorMap, i.portsTCP, i.portsUDP)
	if err != nil {
		return fmt.Errorf("error deploying service '%s': %w", i.k8sName, err)
	}
	i.kubernetesService = service
	logrus.Debugf("Started service '%s'", i.k8sName)
	return nil
}

// patchService patches the service for the instance
func (i *Instance) patchService() error {
	if i.kubernetesService == nil {
		svc, err := k8s.GetService(k8s.Namespace, i.k8sName)
		if err != nil {
			return fmt.Errorf("error getting service '%s': %w", i.k8sName, err)
		}
		i.kubernetesService = svc
	}
	err := k8s.PatchService(k8s.Namespace, i.k8sName, i.kubernetesService.ObjectMeta.Labels, i.kubernetesService.Spec.Selector, i.portsTCP, i.portsUDP)
	if err != nil {
		return fmt.Errorf("error patching service '%s': %w", i.k8sName, err)
	}
	logrus.Debugf("Patched service '%s'", i.k8sName)
	return nil
}

// deployPod deploys the pod for the instance
func (i *Instance) deployPod() error {
	// Get labels for the pod
	labels := i.getLabels()

	// Generate the pod configuration
	podConfig := k8s.PodConfig{
		Namespace: k8s.Namespace,
		Name:      i.k8sName,
		Labels:    labels,
		Image:     i.getTempImageRegistry(), // Get temporary image registry for the pod
		Command:   i.command,
		Args:      i.args,
		Env:       i.env,
		Volumes:   i.volumes,
	}

	// Deploy the pod
	pod, err := k8s.DeployPod(podConfig, true)
	if err != nil {
		return fmt.Errorf("failed to deploy pod: %v", err)
	}

	// Set the state of the instance to started
	i.kubernetesPod = pod

	// Log the deployment of the pod
	logrus.Debugf("Started pod '%s'", i.k8sName)
	logrus.Debugf("Set state of instance '%s' to '%s'", i.k8sName, i.state.String())

	return nil
}

// deployVolume deploys the volume for the instance
func (i *Instance) deployVolume() error {
	size := resource.Quantity{}
	for _, volumeSize := range i.volumes {
		size.Add(resource.MustParse(volumeSize))
	}
	k8s.DeployPersistentVolumeClaim(k8s.Namespace, i.k8sName, i.getLabels(), size, []string{"ReadWriteOnce"})
	logrus.Debugf("Deployed persistent volume '%s'", i.k8sName)

	return nil
}

// destroyVolume destroys the volume for the instance
func (i *Instance) destroyVolume() error {
	k8s.DeletePersistentVolumeClaim(k8s.Namespace, i.k8sName)
	logrus.Debugf("Destroyed persistent volume '%s'", i.k8sName)

	return nil
}

// WaitInstanceIsRunning waits until the instance is running
// This function can only be called in the state 'Started'
func (i *Instance) WaitInstanceIsRunning() error {
	if !i.IsInState(Started) {
		return fmt.Errorf("waiting for instance is only allowed in state 'Started'. Current state is '%s'", i.state.String())
	}
	err := k8s.WaitPodIsRunning(k8s.Namespace, i.k8sName)
	if err != nil {
		return fmt.Errorf("error waiting for pod '%s' is running: %w", i.k8sName, err)
	}

	return nil
}

// Commit commits the instance
// This function can only be called in the state 'Preparing'
func (i *Instance) Commit() error {
	if !i.IsInState(Preparing) {
		return fmt.Errorf("committing is only allowed in state 'Preparing'. Current state is '%s'", i.state.String())
	}
	// TODO: To speed up the process, the image name could be dependent on the hash of the image
	err := container.PushBuilderImage(i.buildContext, i.imageBuilder, i.buildStore, i.getTempImageRegistry())
	if err != nil {
		return fmt.Errorf("error pushing image for instance '%s': %w", i.name, err)
	}
	logrus.Debugf("Pushed image for instance '%s'", i.name)
	i.state = Committed
	logrus.Debugf("Set state of instance '%s' to '%s'", i.name, i.state.String())

	return nil
}

// Start starts the instance
// This function can only be called in the state 'Committed'
func (i *Instance) Start() error {
	if !i.IsInState(Committed) {
		return fmt.Errorf("starting is only allowed in state 'Committed'. Current state is '%s'", i.state.String())
	}
	if len(i.portsTCP) != 0 || len(i.portsUDP) != 0 {
		logrus.Debugf("Ports not empty, deploying service for instance '%s'", i.k8sName)
		svc, _ := k8s.GetService(k8s.Namespace, i.k8sName)
		if svc == nil {
			err := i.deployService()
			if err != nil {
				return fmt.Errorf("error deploying service for instance '%s': %w", i.k8sName, err)
			}
		} else if svc != nil {
			err := i.patchService()
			if err != nil {
				return fmt.Errorf("error patching service for instance '%s': %w", i.k8sName, err)
			}
		}
	}
	if len(i.volumes) != 0 {
		err := i.deployVolume()
		if err != nil {
			return fmt.Errorf("error deploying volume for instance '%s': %w", i.k8sName, err)
		}
	}
	err := i.deployPod()
	if err != nil {
		return fmt.Errorf("error deploying pod for instance '%s': %w", i.k8sName, err)
	}
	i.state = Started
	logrus.Debugf("Set state of instance '%s' to '%s'", i.k8sName, i.state.String())

	return nil
}

// Destroy destroys the instance
// This function can only be called in the state 'Started' or 'Destroyed'
func (i *Instance) Destroy() error {
	if !i.IsInState(Started, Destroyed) {
		return fmt.Errorf("destroying is only allowed in state 'Started' or 'Destroyed'. Current state is '%s'", i.state.String())
	}
	if i.state == Destroyed {
		return nil
	}
	i.destroyPod()
	if len(i.volumes) != 0 {
		err := i.destroyVolume()
		if err != nil {
			return fmt.Errorf("error destroying volume for instance '%s': %w", i.k8sName, err)
		}
	}
	err := i.destroyService()
	if err != nil {
		return fmt.Errorf("error destroying service for instance '%s': %w", i.k8sName, err)
	}

	i.state = Destroyed
	logrus.Debugf("Set state of instance '%s' to '%s'", i.k8sName, i.state.String())

	return nil
}

// destroyService destroys the service for the instance
func (i *Instance) destroyService() error {
	k8s.DeleteService(k8s.Namespace, i.k8sName)

	return nil
}

// destroyPod destroys the pod for the instance
func (i *Instance) destroyPod() error {
	k8s.DeletePod(k8s.Namespace, i.k8sName)

	return nil
}

// getLabels returns the labels for the instance
func (i *Instance) getLabels() map[string]string {
	return map[string]string{"app": i.k8sName}
}

// validatePort validates the port
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port number '%d' is out of range", port)
	}
	return nil
}

// CreatePool creates a pool of instances
// This function can only be called in the state 'Committed'
func (i *Instance) CreatePool(amount int) (*InstancePool, error) {
	if !i.IsInState(Committed) {
		return nil, fmt.Errorf("creating a pool is only allowed in state 'Committed' or 'Destroyed'. Current state is '%s'", i.state.String())
	}
	instances := make([]*Instance, amount)
	for j := 0; j < amount; j++ {
		instances[j] = i.cloneWithSuffix(fmt.Sprintf("-%d", j))
	}

	i.state = Destroyed
	logrus.Debugf("Set state of instance '%s' to '%s'", i.name, i.state.String())

	return &InstancePool{
		instances: instances,
		amount:    amount,
	}, nil
}

// cloneWithSuffix clones the instance with a suffix
func (i *Instance) cloneWithSuffix(suffix string) *Instance {
	return &Instance{
		uuid:              i.uuid,
		name:              i.name + suffix,
		k8sName:           i.k8sName + suffix,
		state:             i.state,
		kubernetesService: i.kubernetesService,
		imageBuilder:      i.imageBuilder,
		buildStore:        i.buildStore,
		buildContext:      i.buildContext,
		kubernetesPod:     i.kubernetesPod,
		portsTCP:          i.portsTCP,
		portsUDP:          i.portsUDP,
		files:             i.files,
		command:           i.command,
		args:              i.args,
		env:               i.env,
		volumes:           i.volumes,
	}
}
