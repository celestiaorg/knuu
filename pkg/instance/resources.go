package instance

import (
	"context"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type resources struct {
	instance      *Instance
	memoryRequest resource.Quantity
	memoryLimit   resource.Quantity
	cpuRequest    resource.Quantity
}

func (i *Instance) Resources() *resources {
	return i.resources
}

// SetMemory sets the memory of the instance
// This function can only be called in the states 'Preparing' and 'Committed'
func (r *resources) SetMemory(request, limit resource.Quantity) error {
	if !r.instance.IsInState(StatePreparing, StateCommitted) {
		return ErrSettingMemoryNotAllowed.WithParams(r.instance.state.String())
	}
	r.memoryRequest = request
	r.memoryLimit = limit
	r.instance.Logger.WithFields(logrus.Fields{
		"instance":       r.instance.name,
		"memory_request": request.String(),
		"memory_limit":   limit.String(),
	}).Debug("set memory for instance")
	return nil
}

// SetCPU sets the CPU of the instance
// This function can only be called in the states 'Preparing' and 'Committed'
func (r *resources) SetCPU(request resource.Quantity) error {
	if !r.instance.IsInState(StatePreparing, StateCommitted) {
		return ErrSettingCPUNotAllowed.WithParams(r.instance.state.String())
	}
	r.cpuRequest = request
	r.instance.Logger.WithFields(logrus.Fields{
		"instance":    r.instance.name,
		"cpu_request": request.String(),
	}).Debug("set cpu for instance")
	return nil
}

// CreateCustomResource creates a custom resource for the instance
// The names and namespace are set and overridden by knuu
func (r *resources) CreateCustomResource(ctx context.Context, gvr *schema.GroupVersionResource, obj *map[string]interface{}) error {
	crdExists, err := r.CustomResourceDefinitionExists(ctx, gvr)
	if err != nil {
		return err
	}
	if !crdExists {
		return ErrCustomResourceDefinitionDoesNotExist.WithParams(gvr.Resource)
	}

	return r.instance.K8sClient.CreateCustomResource(ctx, r.instance.k8sName, gvr, obj)
}

// CustomResourceDefinitionExists checks if the custom resource definition exists
func (r *resources) CustomResourceDefinitionExists(ctx context.Context, gvr *schema.GroupVersionResource) (bool, error) {
	return r.instance.K8sClient.CustomResourceDefinitionExists(ctx, gvr)
}

// deployResources deploys the resources for the instance
func (r *resources) deployResources(ctx context.Context) error {
	// only a non-sidecar instance should deploy a service, all sidecars will use the parent instance's service
	if !r.instance.sidecars.isSidecar {
		if err := r.deployService(ctx); err != nil {
			return err
		}
	}

	if err := r.deployStorage(ctx); err != nil {
		return err
	}
	return nil
}

func (r *resources) deployStorage(ctx context.Context) error {
	if len(r.instance.storage.volumes) != 0 {
		if err := r.instance.storage.deployVolume(ctx); err != nil {
			return ErrDeployingVolumeForInstance.WithParams(r.instance.k8sName).Wrap(err)
		}
	}
	if len(r.instance.storage.files) == 0 {
		return nil
	}

	if err := r.instance.storage.deployFiles(ctx); err != nil {
		return ErrDeployingFilesForInstance.WithParams(r.instance.k8sName).Wrap(err)
	}
	return nil
}

func (r *resources) deployService(ctx context.Context) error {
	portsTCP := r.instance.network.portsTCP
	portsUDP := r.instance.network.portsUDP
	for _, sidecar := range r.instance.sidecars.sidecars {
		portsTCP = append(portsTCP, sidecar.Instance().network.portsTCP...)
		portsUDP = append(portsUDP, sidecar.Instance().network.portsUDP...)
	}
	if len(portsTCP) != 0 || len(portsUDP) != 0 {
		if err := r.instance.network.deployOrPatchService(ctx, portsTCP, portsUDP); err != nil {
			return ErrFailedToDeployOrPatchService.Wrap(err)
		}
	}
	return nil
}

// destroyResources destroys the resources for the instance
func (r *resources) destroyResources(ctx context.Context) error {
	if len(r.instance.storage.volumes) != 0 {
		if err := r.instance.storage.destroyVolume(ctx); err != nil {
			return ErrDestroyingVolumeForInstance.WithParams(r.instance.k8sName).Wrap(err)
		}
	}

	if len(r.instance.storage.files) != 0 {
		err := r.instance.storage.destroyFiles(ctx)
		if err != nil {
			return ErrDestroyingFilesForInstance.WithParams(r.instance.k8sName).Wrap(err)
		}
	}
	if r.instance.network.kubernetesService != nil {
		err := r.instance.network.destroyService(ctx)
		if err != nil {
			return ErrDestroyingServiceForInstance.WithParams(r.instance.k8sName).Wrap(err)
		}
	}

	// disable network only for non-sidecar instances
	if !r.instance.sidecars.IsSidecar() {
		// enable network when network is disabled
		if err := r.instance.network.enableIfDisabled(ctx); err != nil {
			return ErrEnablingNetworkForInstance.WithParams(r.instance.k8sName).Wrap(err)
		}
	}

	return nil
}

func (r *resources) clone() *resources {
	if r == nil {
		return nil
	}

	memoryRequestCopy := r.memoryRequest.DeepCopy()
	memoryLimitCopy := r.memoryLimit.DeepCopy()
	cpuRequestCopy := r.cpuRequest.DeepCopy()

	return &resources{
		instance:      nil,
		memoryRequest: memoryRequestCopy,
		memoryLimit:   memoryLimitCopy,
		cpuRequest:    cpuRequestCopy,
	}
}
