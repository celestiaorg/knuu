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
func (i *InstancePool) Start() {
	for _, instance := range i.instances {
		instance.Start()
	}
}

// Destroy destroys all instances in the instance pool
func (i *InstancePool) Destroy() {
	for _, instance := range i.instances {
		instance.Destroy()
	}
}

// WaitInstancePoolIsRunning waits until all instances in the instance pool are running
func (i *InstancePool) WaitInstancePoolIsRunning() {
	for _, instance := range i.instances {
		instance.WaitInstanceIsRunning()
	}
}
