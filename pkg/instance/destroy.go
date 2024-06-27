package instance

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
)

// Destroy destroys the instance
// This function can only be called in the state 'Started' or 'Destroyed'
func (i *Instance) Destroy(ctx context.Context) error {
	if i.state == Destroyed {
		return nil
	}

	if !i.IsInState(Started, Stopped, Destroyed) {
		return ErrDestroyingNotAllowed.WithParams(i.state.String())
	}

	if err := i.destroyPod(ctx); err != nil {
		return ErrDestroyingPod.WithParams(i.k8sName).Wrap(err)
	}
	if err := i.destroyResources(ctx); err != nil {
		return ErrDestroyingResourcesForInstance.WithParams(i.k8sName).Wrap(err)
	}

	err := applyFunctionToInstances(i.sidecars, func(sidecar Instance) error {
		i.Logger.Debugf("Destroying sidecar resources from '%s'", sidecar.k8sName)
		return sidecar.destroyResources(ctx)
	})
	if err != nil {
		return ErrDestroyingResourcesForSidecars.WithParams(i.k8sName).Wrap(err)
	}

	i.state = Destroyed
	setStateForSidecars(i.sidecars, Destroyed)
	i.Logger.Debugf("Set state of instance '%s' to '%s'", i.k8sName, i.state.String())

	return nil
}

// BatchDestroy destroys a list of instances.
func BatchDestroy(ctx context.Context, instances ...*Instance) error {
	if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
		logrus.Info("Skipping cleanup")
		return nil
	}

	for _, instance := range instances {
		if instance == nil {
			continue
		}
		if err := instance.Destroy(ctx); err != nil {
			return err
		}
	}
	return nil
}
