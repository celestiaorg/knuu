package knuu

import "fmt"

type Executor struct {
	instances *Instance
}

func NewExecutor() (*Executor, error) {
	instance, err := NewInstance("executor")
	if err != nil {
		return nil, fmt.Errorf("error creating instance '%v':", err)
	}
	err = instance.SetImage("docker.io/nicolaka/netshoot:latest")
	if err != nil {
		return nil, fmt.Errorf("error setting image '%v':", err)
	}
	err = instance.Commit()
	if err != nil {
		return nil, fmt.Errorf("error committing instance: %v", err)
	}
	err = instance.SetArgs("sleep", "infinity")
	if err != nil {
		return nil, fmt.Errorf("error setting args '%v':", err)
	}
	err = instance.SetMemory("100M", "100M")
	if err != nil {
		return nil, fmt.Errorf("error setting memory '%v':", err)
	}
	err = instance.SetCPU("100m")
	if err != nil {
		return nil, fmt.Errorf("error setting cpu '%v':", err)
	}
	err = instance.Start()
	if err != nil {
		return nil, fmt.Errorf("error starting instance: %v", err)
	}
	err = instance.WaitInstanceIsRunning()
	if err != nil {
		return nil, fmt.Errorf("error waiting for instance to be running: %v", err)
	}
	return &Executor{
		instances: instance,
	}, nil
}

func (e *Executor) ExecuteCommand(command ...string) (string, error) {
	return e.instances.ExecuteCommand(command...)
}

func (e *Executor) Destroy() error {
	return e.instances.Destroy()
}
