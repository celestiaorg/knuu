package knuu

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// InstancePool is a struct that represents a pool of instances
type InstancePool struct {
	instances []*Instance
	amount    int
}

// Instances returns the instances in the instance pool
func (i *InstancePool) Instances() []*Instance {
	return i.instances
}

// CreatePool creates a pool of instances
// This function can only be called in the state 'Committed'
func (i *Instance) CreatePool(amount int) (*InstancePool, error) {
	if !i.IsInState(Committed) {
		return nil, ErrCreatingPoolNotAllowed.WithParams(i.state.String())
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

// StartWithoutWait starts all instances in the instance pool without waiting for them to be running
func (i *InstancePool) StartWithoutWait() error {
	for _, instance := range i.instances {
		err := instance.StartWithoutWait()
		if err != nil {
			return err
		}
	}
	return nil
}

// Start starts all instances in the instance pool
func (i *InstancePool) Start() error {
	for _, instance := range i.instances {
		err := instance.Start()
		if err != nil {
			return err
		}
	}
	return nil
}

// Destroy destroys all instances in the instance pool
func (i *InstancePool) Destroy() error {
	for _, instance := range i.instances {
		err := instance.Destroy()
		if err != nil {
			return err
		}
	}
	return nil
}

// WaitInstancePoolIsRunning waits until all instances in the instance pool are running
func (i *InstancePool) WaitInstancePoolIsRunning() error {
	for _, instance := range i.instances {
		err := instance.WaitInstanceIsRunning()
		if err != nil {
			return err
		}
	}
	return nil
}
