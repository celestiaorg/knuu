package e2e

import (
	"context"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/celestiaorg/knuu/pkg/knuu"
)

const (
	executorDefaultImage = "docker.io/nicolaka/netshoot:latest"
	sleepCommand         = "sleep"
	infinityArg          = "infinity"
)

var (
	executorMemoryLimit = resource.MustParse("100Mi")
	executorCpuLimit    = resource.MustParse("100m")
)

func NewExecutor(ctx context.Context, executorName string) (*knuu.Instance, error) {
	i, err := knuu.NewInstance(executorName)
	if err != nil {
		return nil, err
	}

	if err := i.SetImage(executorDefaultImage); err != nil {
		return nil, err
	}

	if err := i.Commit(); err != nil {
		return nil, err
	}

	if err := i.SetArgs(sleepCommand, infinityArg); err != nil {
		return nil, err
	}

	if err := i.SetMemory(executorMemoryLimit.String(), executorMemoryLimit.String()); err != nil {
		return nil, err
	}

	if err := i.SetCPU(executorCpuLimit.String()); err != nil {
		return nil, err
	}

	if err := i.Start(); err != nil {
		return nil, err
	}

	return i, nil
}
