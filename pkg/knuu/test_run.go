package knuu

// TestRun is an interface that defines the methods that a test run should implement
type TestRun interface {
	Instances() []*Instance
	InstancePools() []*InstancePool
	Prepare() error
	Test() error
	Clean() error
}

// BaseTestRun is a struct that implements the TestRun interface
type BaseTestRun struct {
	instanceArray     []*Instance
	instancePoolArray []*InstancePool
}

// Instances returns the instances that are part of the test run
func (b *BaseTestRun) Instances() []*Instance {
	return b.instanceArray
}

// InstancePools returns the instance pools that are part of the test run
func (b *BaseTestRun) InstancePools() []*InstancePool {
	return b.InstancePools()
}

// AddInstance adds an instance to the test run
func (b *BaseTestRun) AddInstance(instance *Instance) {
	b.instanceArray = append(b.instanceArray, instance)
}

// AddInstancePool adds an instance pool to the test run
func (b *BaseTestRun) AddInstancePool(instancePool *InstancePool) {
	b.instancePoolArray = append(b.instancePoolArray, instancePool)
}

// Clean cleans up after the test run
func (b *BaseTestRun) Clean() error {
	for _, instance := range b.instanceArray {
		instance.Destroy()
	}
	for _, instancePool := range b.instancePoolArray {
		instancePool.Destroy()
	}
	return nil
}
