// Package knuu provides the core functionality of knuu.
package knuu

import (
	"context"
	"fmt"
	"io/ioutil"
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

// Instance represents a container instance
type Instance struct {
	uuid              uuid.UUID
	name              string
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
func NewInstance(name string) *Instance {
	// Generate a UUID for this container
	uuid, err := uuid.NewRandom()
	if err != nil {
		logrus.Fatalf("error generating UUID for container '%s': %s", name, err.Error())
	}
	// Create the instance
	return &Instance{
		uuid:     uuid,
		name:     name,
		state:    None,
		portsTCP: make([]int, 0),
		portsUDP: make([]int, 0),
		files:    make([]string, 0),
		command:  make([]string, 0),
		args:     make([]string, 0),
		env:      make(map[string]string),
		volumes:  make(map[string]string),
	}
}

// isInState checks if the instance is in one of the provided states
func (i *Instance) isInState(states ...State) bool {
	for _, s := range states {
		if i.state == s {
			return true
		}
	}
	return false
}

// SetImage sets the image of the instance
// Only allowed in state 'None' and 'Started'
func (i *Instance) SetImage(image string) {
	// Only allow setting the image when the instance is in the 'None' or 'Started' state
	if !i.isInState(None, Started) {
		logrus.Fatalf("Setting image is only allowed in state 'None' and 'Started'. Current state is '%s'", i.state.String())
	}

	if i.state == None {
		// Create a new build context
		context, _ := context.WithCancel(context.Background())

		i.buildContext = context

		// Use the builder to build a new image
		builder, storage, err := container.NewBuilder(context, image)
		if err != nil {
			logrus.Fatalf("Error getting builder: %s", err.Error())
		}
		i.imageBuilder = builder
		i.buildStore = storage
		i.state = Preparing
	} else if i.state == Started {
		// Replace the pod with a new one, using the given image
		k8s.ReplacePod(k8s.Namespace, i.name, i.kubernetesPod.Labels, image, i.command, i.args, i.env, i.volumes)
		i.WaitInstanceIsRunning()
	}
}

// SetCommand sets the command to run in the container
// This function can only be called when the instance is in state 'Preparing' or 'Committed'
func (i *Instance) SetCommand(command []string) error {
	if !i.isInState(Preparing, Committed) {
		logrus.Fatalf("Setting command is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	i.command = command
	return nil
}

// SetArgs sets the arguments passed to the instance
// This function can only be called in the states 'Preparing' or 'Committed'
func (i *Instance) SetArgs(args []string) error {
	if !i.isInState(Preparing, Committed) {
		logrus.Fatalf("Setting args is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
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
func (i *Instance) AddPortTCP(port int) {
	if !i.isInState(Preparing, Committed) {
		logrus.Fatalf("Adding port is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	validatePort(port)
	if i.isTCPPortRegistered(port) {
		logrus.Fatalf("TCP port '%d' is already in registered", port)
	}
	i.portsTCP = append(i.portsTCP, port)
	logrus.Debugf("Added TCP port '%d' to container '%s'", port, i.name)
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
func (i *Instance) AddPortUDP(port int) {
	if !i.isInState(Preparing, Committed) {
		logrus.Fatalf("Adding port is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	validatePort(port)
	if i.isUDPPortRegistered(port) {
		logrus.Fatalf("UDP port '%d' is already in registered", port)
	}
	i.portsUDP = append(i.portsUDP, port)
	logrus.Debugf("Added UDP port '%d' to container '%s'", port, i.name)
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
func (i *Instance) ExecuteCommand(command []string) string {
	if !i.isInState(Preparing, Started) {
		logrus.Fatalf("Executing command is only allowed in state 'Preparing' or 'Started'. Current state is '%s'", i.state.String())
	}
	if i.isInState(Preparing) {
		output, err := container.ExecuteCmdInBuilder(i.imageBuilder, command)
		if err != nil {
			logrus.Fatalf("Error executing command '%s' in container '%s': %s", command, i.name, err)
		}
		return output
	} else if i.isInState(Started) {
		output, err := k8s.RunCommandInPod(k8s.Namespace, i.name, i.name, command)
		if err != nil {
			logrus.Fatalf("Error executing command '%s' in started container '%s': %s", command, i.name, err)
		}
		return output
	} else {
		logrus.Fatalf("Cannot execute command '%s' in container '%s' in state '%s'", command, i.name, i.state.String())
	}

	return ""
}

// AddFileBytes adds a file with the given content to the instance
// This function can only be called in the state 'Preparing'
func (i *Instance) AddFileBytes(bytes []byte, dest string, chown string) {
	if !i.isInState(Preparing) {
		logrus.Fatalf("Adding file is only allowed in state 'Preparing'. Current state is '%s'", i.state.String())
	}

	tmpFile, err := ioutil.TempFile("", "temp-file-")
	if err != nil {
		logrus.Fatalf("Error creating temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(bytes)
	if err != nil {
		logrus.Fatalf("Error writing content to temporary file: %w", err)
	}
	tmpFile.Close()

	i.AddFile(tmpFile.Name(), dest, chown)
}

// AddFile adds a file to the instance
// This function can only be called in the state 'Preparing'
func (i *Instance) AddFile(src string, dest string, chown string) {
	if !i.isInState(Preparing) {
		logrus.Fatalf("Adding file is only allowed in state 'Preparing'. Current state is '%s'", i.state.String())
	}

	if i.imageBuilder == nil {
		logrus.Fatalf("Image not set for container '%s'. Set it with setImage(image string)", i.name)
	}
	i.files = append(i.files, dest)
	err := container.AddFileToBuilder(i.imageBuilder, src, dest, chown)
	if err != nil {
		logrus.Fatalf("Error adding file '%s' to container '%s': %w", dest, i.name, err)
	}
	logrus.Debugf("Added file '%s' to container '%s'", dest, i.name)
}

// GetFileBytes returns the content of the given file
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) GetFileBytes(file string) []byte {
	if !i.isInState(Preparing, Committed) {
		logrus.Fatalf("Getting file is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}

	bytes, err := container.ReadFileFromBuilder(i.imageBuilder, file)
	if err != nil {
		logrus.Fatalf("Error getting file '%s' from container '%s': %w", file, i.name, err)
	}
	return bytes
}

// SetEnvironmentVariable sets the given environment variable in the instance
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) SetEnvironmentVariable(key string, value string) {
	if !i.isInState(Preparing, Committed) {
		logrus.Fatalf("Setting environment variable is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	if i.state == Preparing {
		container.SetEnvVar(i.imageBuilder, key, value)
	} else if i.state == Committed {
		i.env[key] = value
	}
	logrus.Debugf("Set environment variable '%s' to '%s' in container '%s'", key, value, i.name)
}

// AddVolume adds a volume to the instance
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) AddVolume(name string, size string) {
	if !i.isInState(Preparing, Committed) {
		logrus.Fatalf("Adding volume is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	i.volumes[name] = size
}

// GetIP returns the IP of the instance
// This function can only be called in the states 'Preparing' and 'Started'
func (i *Instance) GetIP() string {
	if !k8s.ServiceExists(k8s.Namespace, i.name) {
		i.deployService()
	}

	return k8s.GetServiceIP(k8s.Namespace, i.name)
}

// deployService deploys the service for the instance
func (i *Instance) deployService() {
	if k8s.ServiceExists(k8s.Namespace, i.name) {
		i.patchService()
	}
	labels := i.getLabels()
	selectorMap := i.getLabels()
	i.kubernetesService = k8s.DeployService(k8s.Namespace, i.name, labels, selectorMap, i.portsTCP, i.portsUDP)
	logrus.Debugf("Started service '%s'", i.name)
}

// patchService patches the service for the instance
func (i *Instance) patchService() {
	if i.kubernetesService == nil {
		i.kubernetesService = k8s.GetService(k8s.Namespace, i.name)
	}
	k8s.PatchService(k8s.Namespace, i.name, i.kubernetesService.ObjectMeta.Labels, i.kubernetesService.Spec.Selector, i.portsTCP, i.portsUDP)
	logrus.Debugf("Patched service '%s'", i.name)
}

// deployPod deploys the pod for the instance
func (i *Instance) deployPod() {
	labels := i.getLabels()
	i.kubernetesPod = k8s.DeployPod(k8s.Namespace, i.name, labels, i.getTempImageRegistry(), i.command, i.args, i.env, i.volumes, true)
	logrus.Debugf("Started pod '%s'", i.name)
	i.state = Started
	logrus.Debugf("Set state of container '%s' to '%s'", i.name, i.state.String())
}

// deployVolume deploys the volume for the instance
func (i *Instance) deployVolume() {
	size := resource.Quantity{}
	for _, volumeSize := range i.volumes {
		size.Add(resource.MustParse(volumeSize))
	}
	k8s.DeployPersistentVolumeClaim(k8s.Namespace, i.name, i.getLabels(), size, []string{"ReadWriteOnce"})
	logrus.Debugf("Deployed persistent volume '%s'", i.name)
}

// destroyVolume destroys the volume for the instance
func (i *Instance) destroyVolume() {
	k8s.DeletePersistentVolumeClaim(k8s.Namespace, i.name)
	logrus.Debugf("Destroyed persistent volume '%s'", i.name)
}

// WaitInstanceIsRunning waits until the instance is running
// This function can only be called in the state 'Started'
func (i *Instance) WaitInstanceIsRunning() {
	if !i.isInState(Started) {
		logrus.Fatalf("Waiting for instance is only allowed in state 'Started'. Current state is '%s'", i.state.String())
	}
	k8s.WaitPodIsRunning(k8s.Namespace, i.name)
}

// Commit commits the instance
// This function can only be called in the state 'Preparing'
func (i *Instance) Commit() {
	if !i.isInState(Preparing) {
		logrus.Fatalf("Committing is only allowed in state 'Preparing'. Current state is '%s'", i.state.String())
	}
	// TODO: To speed up the process, the image name could be dependent on the hash of the image
	err := container.PushBuilderImage(i.buildContext, i.imageBuilder, i.buildStore, i.getTempImageRegistry())
	if err != nil {
		logrus.Fatalf("Error pushing image for container '%s': %s", i.name, err)
	}
	logrus.Debugf("Pushed image for container '%s'", i.name)
	i.state = Committed
	logrus.Debugf("Set state of container '%s' to '%s'", i.name, i.state.String())
}

// Start starts the instance
// This function can only be called in the state 'Committed'
func (i *Instance) Start() {
	if !i.isInState(Committed) {
		logrus.Fatalf("Starting is only allowed in state 'Committed'. Current state is '%s'", i.state.String())
	}
	if len(i.portsTCP) != 0 || len(i.portsUDP) != 0 {
		logrus.Debugf("Ports not empty, deploying service for container '%s'", i.name)
		if !k8s.ServiceExists(k8s.Namespace, i.name) {
			i.deployService()
		} else {
			i.patchService()
		}
	}
	if len(i.volumes) != 0 {
		i.deployVolume()
	}
	i.deployPod()
}

// Destroy destroys the instance
// This function can only be called in the state 'Started' or 'Destroyed'
func (i *Instance) Destroy() {
	if !i.isInState(Started, Destroyed) {
		logrus.Fatalf("Destroying is only allowed in state 'Started' or 'Destroyed'. Current state is '%s'", i.state.String())
	}
	if i.state == Destroyed {
		return
	}
	i.destroyPod()
	if len(i.volumes) != 0 {
		i.destroyVolume()
	}
	i.destroyService()

	i.state = Destroyed
}

// destroyService destroys the service for the instance
func (i *Instance) destroyService() {
	k8s.DeleteService(k8s.Namespace, i.name)
}

// destroyPod destroys the pod for the instance
func (i *Instance) destroyPod() {
	k8s.DeletePod(k8s.Namespace, i.name)
}

// getLabels returns the labels for the instance
func (i *Instance) getLabels() map[string]string {
	return map[string]string{"app": i.name}
}

// validatePort validates the port
func validatePort(port int) {
	if port < 1 || port > 65535 {
		logrus.Fatalf("Port number '%d' is out of range", port)
	}
}

// CreatePool creates a pool of instances
// This function can only be called in the state 'Committed'
func (i *Instance) CreatePool(amount int) *InstancePool {
	if !i.isInState(Committed) {
		logrus.Fatalf("Creating a pool is only allowed in state 'Committed' or 'Destroyed'. Current state is '%s'", i.state.String())
	}
	instances := make([]*Instance, amount)
	for j := 0; j < amount; j++ {
		instances[j] = i.cloneWithSuffix(fmt.Sprintf("-%d", j))
	}

	i.state = Destroyed

	return &InstancePool{
		instances: instances,
		amount:    amount,
	}
}

// cloneWithSuffix clones the instance with a suffix
func (i *Instance) cloneWithSuffix(suffix string) *Instance {
	return &Instance{
		uuid:              i.uuid,
		name:              i.name + suffix,
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
