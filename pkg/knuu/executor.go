package knuu

import (
	"context"
)

type Executor struct {
	instances *Instance
}

func NewExecutor() (*Executor, error) {
	instance, err := NewInstance("executor")
	if err != nil {
		return nil, ErrCreatingInstance.Wrap(err)
	}
	err = instance.SetImage("docker.io/nicolaka/netshoot:latest")
	if err != nil {
		return nil, ErrSettingImage.Wrap(err)
	}
	err = instance.Commit()
	if err != nil {
		return nil, ErrCommittingInstance.Wrap(err)
	}
	err = instance.SetArgs("sleep", "infinity")
	if err != nil {
		return nil, ErrSettingArgs.Wrap(err)
	}
	err = instance.SetMemory("100M", "100M")
	if err != nil {
		return nil, ErrSettingMemory.Wrap(err)
	}
	err = instance.SetCPU("100m")
	if err != nil {
		return nil, ErrSettingCPU.Wrap(err)
	}
	instance.instanceType = ExecutorInstance
	err = instance.Start()
	if err != nil {
		return nil, ErrStartingInstance.Wrap(err)
	}
	err = instance.WaitInstanceIsRunning()
	if err != nil {
		return nil, ErrWaitingInstanceIsRunning.Wrap(err)
	}
	return &Executor{
		instances: instance,
	}, nil
}

func (e *Executor) ExecuteCommand(command ...string) (string, error) {
	return e.instances.ExecuteCommand(command...)
}

func (e *Executor) ExecuteCommandWithContext(ctx context.Context, command ...string) (string, error) {
	return e.instances.ExecuteCommandWithContext(ctx, command...)
}

func (e *Executor) Destroy() error {
	return e.instances.Destroy()
}
