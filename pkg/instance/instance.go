package instance

import (
	"sync"
	"time"

	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/names"
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
	system.SystemDependencies

	resources  *resources
	network    *network
	build      *build
	execution  *execution
	storage    *storage
	monitoring *monitoring
	security   *security
	sidecars   *sidecars

	name         string
	k8sName      string
	state        InstanceState
	instanceType InstanceType

	kubernetesReplicaSet *appv1.ReplicaSet

	parentInstance *Instance
}

func New(name string, sysDeps system.SystemDependencies) (*Instance, error) {
	k8sName, err := names.NewRandomK8(name)
	if err != nil {
		return nil, ErrGeneratingK8sName.WithParams(name).Wrap(err)
	}

	// Create the instance
	i := &Instance{
		name:               name,
		k8sName:            k8sName,
		state:              StateNone,
		instanceType:       BasicInstance,
		SystemDependencies: sysDeps,
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

func (i *Instance) K8sName() string {
	return i.k8sName
}

func (i *Instance) State() InstanceState {
	return i.state
}

func (i *Instance) SetInstanceType(instanceType InstanceType) {
	i.instanceType = instanceType
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
	ins := i.CloneWithSuffix("")
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
	ins := i.CloneWithSuffix("")
	ins.name = name
	ins.k8sName = newK8sName
	return ins, nil
}

// cloneWithSuffix clones the instance with a suffix
func (i *Instance) CloneWithSuffix(suffix string) *Instance {
	newInstance := &Instance{
		name:               i.name + "-" + suffix,
		k8sName:            i.k8sName + "-" + suffix,
		SystemDependencies: i.SystemDependencies,

		build:      i.build.clone(),
		execution:  i.execution.clone(),
		resources:  i.resources.clone(),
		network:    i.network.clone(),
		storage:    i.storage.clone(),
		monitoring: i.monitoring.clone(),
		security:   i.security.clone(),
		sidecars:   i.sidecars.cloneWithSuffix(suffix),

		state:        i.state,
		instanceType: i.instanceType,
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

	return newInstance
}
