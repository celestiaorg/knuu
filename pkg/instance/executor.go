package instance

import (
	"context"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/celestiaorg/knuu/pkg/system"
)

const (
	executorDefaultImage = "docker.io/nicolaka/netshoot:latest"
	executorName         = "executor"
	sleepCommand         = "sleep"
	infinityArg          = "infinity"
	memoryLimit          = "100M"
	cpuLimit             = "100m"
)

type Executor struct {
	*Instance
}

func NewExecutor(ctx context.Context, sysDeps system.SystemDependencies) (*Executor, error) {
	i, err := New(executorName, sysDeps)
	if err != nil {
		return nil, ErrCreatingInstance.Wrap(err)
	}

	if err := i.SetImage(ctx, executorDefaultImage); err != nil {
		return nil, ErrSettingImage.Wrap(err)
	}

	if err := i.Commit(); err != nil {
		return nil, ErrCommittingInstance.Wrap(err)
	}

	if err := i.SetArgs(sleepCommand, infinityArg); err != nil {
		return nil, ErrSettingArgs.Wrap(err)
	}

	if err := i.SetMemory(resource.MustParse(memoryLimit), resource.MustParse(memoryLimit)); err != nil {
		return nil, ErrSettingMemory.Wrap(err)
	}

	if err := i.SetCPU(resource.MustParse(cpuLimit)); err != nil {
		return nil, ErrSettingCPU.Wrap(err)
	}
	i.instanceType = ExecutorInstance

	if err := i.Start(ctx); err != nil {
		return nil, ErrStartingInstance.Wrap(err)
	}

	return &Executor{Instance: i}, nil
}
