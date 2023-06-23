package knuu

import (
	"fmt"
	"github.com/celestiaorg/knuu/pkg/container"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/sirupsen/logrus"
	"io"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"os"
	"path/filepath"
	"time"
)

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
}

// NewInstance creates a new instance of the Instance struct
func NewInstance(name string) (*Instance, error) {
	// Generate a UUID for this instance

	k8sName, err := generateK8sName(name)
	if err != nil {
		return nil, fmt.Errorf("error generating k8s name for instance '%s': %w", name, err)
	}
	// Create the instance
	return &Instance{
		name:          name,
		k8sName:       k8sName,
		imageName:     "",
		state:         None,
		instanceType:  BasicInstance,
		portsTCP:      make([]int, 0),
		portsUDP:      make([]int, 0),
		command:       make([]string, 0),
		args:          make([]string, 0),
		env:           make(map[string]string),
		volumes:       make([]*k8s.Volume, 0),
		memoryRequest: "",
		memoryLimit:   "",
		cpuRequest:    "",
	}, nil
}

// SetImage sets the image of the instance.
// When calling in state 'Started', make sure to call AddVolume() before.
// It is only allowed in the 'None' and 'Started' states.
func (i *Instance) SetImage(image string) error {
	// Check if setting the image is allowed in the current state
	if !i.IsInState(None, Started) {
		return fmt.Errorf("setting image is only allowed in state 'None' and 'Started'. Current state is '%s'", i.state.String())
	}

	var err error

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

		// Generate the pod configuration
		podConfig := k8s.PodConfig{
			Namespace:     k8s.Namespace(),
			Name:          i.k8sName,
			Labels:        i.kubernetesStatefulSet.Labels,
			Image:         image,
			Command:       i.command,
			Args:          i.args,
			Env:           i.env,
			Volumes:       i.volumes,
			MemoryRequest: i.memoryRequest,
			MemoryLimit:   i.memoryLimit,
			CPURequest:    i.cpuRequest,
		}
		// Generate the statefulset configuration
		statefulSetConfig := k8s.StatefulSetConfig{
			Namespace: k8s.Namespace(),
			Name:      i.k8sName,
			Labels:    i.kubernetesStatefulSet.Labels,
			Replicas:  1,
			PodConfig: podConfig,
		}

		// Replace the pod with a new one, using the given image
		_, err = k8s.ReplaceStatefulSet(statefulSetConfig)
		if err != nil {
			return fmt.Errorf("error replacing pod: %s", err.Error())
		}
		i.WaitInstanceIsRunning()
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

	// Generate the pod configuration
	podConfig := k8s.PodConfig{
		Namespace:     k8s.Namespace(),
		Name:          i.k8sName,
		Labels:        i.kubernetesStatefulSet.Labels,
		Image:         image,
		Command:       i.command,
		Args:          i.args,
		Env:           i.env,
		Volumes:       i.volumes,
		MemoryRequest: i.memoryRequest,
		MemoryLimit:   i.memoryLimit,
		CPURequest:    i.cpuRequest,
	}
	// Generate the statefulset configuration
	statefulSetConfig := k8s.StatefulSetConfig{
		Namespace: k8s.Namespace(),
		Name:      i.k8sName,
		Labels:    i.kubernetesStatefulSet.Labels,
		Replicas:  1,
		PodConfig: podConfig,
	}

	// Replace the pod with a new one, using the given image
	gracePeriod := int64(1)
	_, err := k8s.ReplaceStatefulSetWithGracePeriod(statefulSetConfig, &gracePeriod)
	if err != nil {
		return fmt.Errorf("error replacing pod: %s", err.Error())
	}
	i.WaitInstanceIsRunning()

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
	validatePort(port)
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
	validatePort(port)
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
	err = k8s.PortForwardPod(k8s.Namespace(), pod.Name, localPort, port)
	if err != nil {
		return -1, fmt.Errorf("error forwarding port: %v", err)
	}
	return localPort, nil
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

// ExecuteCommand executes the given command in the instance
// This function can only be called in the states 'Preparing' and 'Started'
func (i *Instance) ExecuteCommand(command ...string) (string, error) {
	if !i.IsInState(Preparing, Started) {
		return "", fmt.Errorf("executing command is only allowed in state 'Preparing' or 'Started'. Current state is '%s'", i.state.String())
	}
	if i.IsInState(Preparing) {
		output, err := i.builderFactory.ExecuteCmdInBuilder(command)
		if err != nil {
			return "", fmt.Errorf("error executing command '%s' in instance '%s': %v", command, i.name, err)
		}
		return output, nil
	} else if i.IsInState(Started) {
		pod, err := k8s.GetFirstPodFromStatefulSet(k8s.Namespace(), i.k8sName)
		if err != nil {
			return "", fmt.Errorf("error getting pod from statefulset '%s': %v", i.k8sName, err)
		}
		output, err := k8s.RunCommandInPod(k8s.Namespace(), pod.Name, i.k8sName, command)
		if err != nil {
			return "", fmt.Errorf("error executing command '%s' in started instance '%s': %v", command, i.k8sName, err)
		}
		return output, nil
	} else {
		return "", fmt.Errorf("cannot execute command '%s' in instance '%s' in state '%s'", command, i.k8sName, i.state.String())
	}

	return "", nil
}

// AddFile adds a file to the instance
// This function can only be called in the state 'Preparing'
func (i *Instance) AddFile(src string, dest string, chown string) error {
	if !i.IsInState(Preparing) {
		return fmt.Errorf("adding file is only allowed in state 'Preparing'. Current state is '%s'", i.state.String())
	}

	i.validateFileArgs(src, dest, chown)

	// check if src exists (either as file or as folder)
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("src '%s' does not exist", src)
	}

	// copy file to build dir
	dstPath := filepath.Join(i.getBuildDir(), dest)

	// make sure dir exists
	err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm)
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

	i.addFileToBuilder(src, dest, chown)

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
	if !i.IsInState(Preparing) {
		return fmt.Errorf("adding file is only allowed in state 'Preparing'. Current state is '%s'", i.state.String())
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
func (i *Instance) AddVolume(path string, size string) error {
	i.AddVolumeWithOwner(path, size, 0)
	return nil
}

// AddVolumeWithOwner adds a volume to the instance with the given owner
// This function can only be called in the states 'Preparing' and 'Committed'
func (i *Instance) AddVolumeWithOwner(path string, size string, owner int64) error {
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
func (i *Instance) SetMemory(request string, limit string) error {
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
func (i *Instance) SetEnvironmentVariable(key string, value string) error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("setting environment variable is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", i.state.String())
	}
	if i.state == Preparing {
		i.builderFactory.SetEnvVar(key, value)
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

// Start starts the instance
// This function can only be called in the state 'Committed'
func (i *Instance) Start() error {
	if !i.IsInState(Committed, Stopped) {
		return fmt.Errorf("starting is only allowed in state 'Committed'. Current state is '%s'", i.state.String())
	}
	if i.state == Committed {
		if len(i.portsTCP) != 0 || len(i.portsUDP) != 0 {
			logrus.Debugf("Ports not empty, deploying service for instance '%s'", i.k8sName)
			svc, _ := k8s.GetService(k8s.Namespace(), i.k8sName)
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
	}
	err := i.deployPod()
	if err != nil {
		return fmt.Errorf("error deploying pod for instance '%s': %w", i.k8sName, err)
	}
	i.state = Started
	logrus.Debugf("Set state of instance '%s' to '%s'", i.k8sName, i.state.String())

	err = i.WaitInstanceIsRunning()
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

	return nil
}

// DisableNetwork disables the network of the instance
// This does not apply to executor instances
// This function can only be called in the state 'Started'
func (i *Instance) DisableNetwork() error {
	if !i.IsInState(Started) {
		return fmt.Errorf("disabling network is only allowed in state 'Started'. Current state is '%s'", i.state.String())
	}
	executorSelectorMap := map[string]string{
		"type": ExecutorInstance.String(),
	}
	err := k8s.CreateNetworkPolicy(k8s.Namespace(), i.k8sName, i.getLabels(), executorSelectorMap, executorSelectorMap)
	if err != nil {
		return fmt.Errorf("error disabling network for instance '%s': %w", i.k8sName, err)
	}
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
	if len(i.volumes) != 0 {
		err := i.destroyVolume()
		if err != nil {
			return fmt.Errorf("error destroying volume for instance '%s': %w", i.k8sName, err)
		}
	}
	err = i.destroyService()
	if err != nil {
		return fmt.Errorf("error destroying service for instance '%s': %w", i.k8sName, err)
	}

	i.state = Destroyed
	logrus.Debugf("Set state of instance '%s' to '%s'", i.k8sName, i.state.String())

	return nil
}

// Clone creates a clone of the instance
// This function can only be called in the state 'Committed'
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
