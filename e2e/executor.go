package e2e

import (
	"context"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/knuu"
)

const (
	executorDefaultImage = "docker.io/nicolaka/netshoot:latest"
	sleepCommand         = "sleep"
	infinityArg          = "infinity"
)

type Executor struct {
	Kn *knuu.Knuu
}

var (
	executorMemoryLimit = resource.MustParse("100Mi")
	executorCpuLimit    = resource.MustParse("100m")
)

func (e *Executor) NewInstance(ctx context.Context, name string) (*instance.Instance, error) {
	i, err := e.Kn.NewInstance(name)
	if err != nil {
		return nil, err
	}

	if err := i.SetImage(ctx, executorDefaultImage); err != nil {
		return nil, err
	}

	if err := i.Commit(); err != nil {
		return nil, err
	}

	if err := i.SetArgs(sleepCommand, infinityArg); err != nil {
		return nil, err
	}

	if err := i.SetMemory(executorMemoryLimit, executorMemoryLimit); err != nil {
		return nil, err
	}

	if err := i.SetCPU(executorCpuLimit); err != nil {
		return nil, err
	}

	if err := i.Start(ctx); err != nil {
		return nil, err
	}

	return i, nil
}
