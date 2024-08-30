package instance

import (
	"sync"
	"time"

	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/system"
)

// We need to retry here because the port forwarding might fail as getFreePortTCP() might not free the port fast enough
const (
	maxRetries           = 5
	retryInterval        = 5 * time.Second
	waitForInstanceRetry = 1 * time.Second
	labelType            = "knuu.sh/type"
)

// Instance represents a instance
type Instance struct {
	*system.SystemDependencies

	resources  *resources
	network    *network
	build      *build
	execution  *execution
	storage    *storage
	monitoring *monitoring
	security   *security
	sidecars   *sidecars

	name         string
	state        InstanceState
	instanceType InstanceType

	kubernetesReplicaSet *appv1.ReplicaSet

	parentInstance *Instance
}

func New(name string, sysDeps *system.SystemDependencies) (*Instance, error) {
	i := &Instance{
		state:              StateNone,
		instanceType:       BasicInstance,
		SystemDependencies: sysDeps,
	}

	if err := i.SetName(name); err != nil {
		return nil, err
	}

	i.build = &build{
		instance:        i,
		command:         make([]string, 0),
		args:            make([]string, 0),
		env:             make(map[string]string),
		imageCache:      &sync.Map{},
		imagePullPolicy: v1.PullAlways,
	}

	i.execution = &execution{instance: i}
	i.resources = &resources{
		instance:      i,
		memoryRequest: resource.Quantity{},
		memoryLimit:   resource.Quantity{},
		cpuRequest:    resource.Quantity{},
	}
	i.network = &network{
		instance: i,
		portsTCP: make([]int, 0),
		portsUDP: make([]int, 0),
	}
	i.storage = &storage{
		instance: i,
		volumes:  make([]*k8s.Volume, 0),
		files:    make([]*k8s.File, 0),
	}

	i.monitoring = &monitoring{
		instance:       i,
		livenessProbe:  nil,
		readinessProbe: nil,
		startupProbe:   nil,
	}

	i.security = &security{
		instance:        i,
		privileged:      false,
		capabilitiesAdd: make([]string, 0),
		policyRules:     make([]rbacv1.PolicyRule, 0),
	}

	i.sidecars = &sidecars{
		instance: i,
		sidecars: make([]SidecarManager, 0),
	}

	return i, nil
}

func (i *Instance) Name() string {
	return i.name
}

func (i *Instance) SetName(name string) error {
	name = k8s.SanitizeName(name)
	if i.SystemDependencies.HasInstanceName(name) {
		return ErrInstanceNameAlreadyExists.WithParams(name)
	}
	i.SystemDependencies.AddInstanceName(name)

	if i.name != "" {
		// Remove the old name from the system dependencies
		// So someone else can use it if they want
		i.SystemDependencies.RemoveInstanceName(i.name)
	}
	i.name = name
	return nil
}

func (i *Instance) State() InstanceState {
	return i.state
}

func (i *Instance) SetInstanceType(instanceType InstanceType) {
	i.instanceType = instanceType
}

// cloneWithSuffix clones the instance with a suffix
func (i *Instance) CloneWithSuffix(suffix string) (*Instance, error) {
	return i.CloneWithName(i.name + "-" + suffix)
}

// CloneWithName creates a clone of the instance with a given name
// This function can only be called in the state 'Committed'
// When cloning an instance that is a sidecar, the clone will be not a sidecar
// When cloning an instance with sidecars, the sidecars will be cloned as well
func (i *Instance) CloneWithName(name string) (*Instance, error) {
	clonedSidecars, err := i.sidecars.clone(name)
	if err != nil {
		return nil, err
	}

	newInstance := &Instance{
		SystemDependencies: i.SystemDependencies,

		build:      i.build.clone(),
		execution:  i.execution.clone(),
		resources:  i.resources.clone(),
		network:    i.network.clone(),
		storage:    i.storage.clone(),
		monitoring: i.monitoring.clone(),
		security:   i.security.clone(),
		sidecars:   clonedSidecars,

		state:        i.state,
		instanceType: i.instanceType,
	}

	if err := newInstance.SetName(name); err != nil {
		return nil, err
	}

	// Need to set all the parent references to the newly created instance
	newInstance.sidecars.instance = newInstance
	newInstance.security.instance = newInstance
	newInstance.monitoring.instance = newInstance
	newInstance.storage.instance = newInstance
	newInstance.network.instance = newInstance
	newInstance.execution.instance = newInstance
	newInstance.resources.instance = newInstance
	newInstance.build.instance = newInstance

	return newInstance, nil
}
