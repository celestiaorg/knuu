// Package knuu provides the core functionality of knuu.
package knuu

// InstancePool is a struct that represents a pool of instances
type InstancePool struct {
	instances []*Instance
	amount    int
}

// Instances returns the instances in the instance pool
func (i *InstancePool) Instances() []*Instance {
	return i.instances
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
