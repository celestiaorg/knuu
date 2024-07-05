package instance

import (
	"context"
	"fmt"
	"time"

	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

const (
	chaosMeshAPIGroupVersion = "chaos-mesh.org/v1alpha1"
	chaosMeshAPIVersion      = "v1alpha1"
	chaosMeshAPIGroup        = "chaos-mesh.org"
	chaosMeshNetworkResource = "networkchaos"
)

func (i *Instance) EnableChaosMesh() error {
	if !i.IsChaosMeshEnabled() {
		return ErrChaosMeshNotEnabledInKnuu
	}

	i.gvr = schema.GroupVersionResource{
		Group:    chaosMeshAPIGroup,
		Version:  chaosMeshAPIVersion,
		Resource: chaosMeshNetworkResource,
	}

	return nil
}

// SetDelay sets a delay on the network traffic
func (i *Instance) SetDelay(ctx context.Context, delay, duration time.Duration) error {
	tcParam := v1alpha1.TcParameter{
		Delay: &v1alpha1.DelaySpec{
			Latency: delay.String(),
		}}
	return i.setNetworkChaos(ctx, v1alpha1.DelayAction, tcParam, duration)
}

// SetLoss sets packet loss on the network traffic
func (i *Instance) SetLoss(ctx context.Context, loss float32, duration time.Duration) error {
	tcParam := v1alpha1.TcParameter{
		Loss: &v1alpha1.LossSpec{
			Loss: fmt.Sprintf("%f", loss),
		},
	}
	return i.setNetworkChaos(ctx, v1alpha1.LossAction, tcParam, duration)
}

// SetDuplicate sets packet duplication on the network traffic
func (i *Instance) SetDuplicate(ctx context.Context, duplicate float32, duration time.Duration) error {
	tcParam := v1alpha1.TcParameter{
		Duplicate: &v1alpha1.DuplicateSpec{
			Duplicate: fmt.Sprintf("%f", duplicate),
		},
	}
	return i.setNetworkChaos(ctx, v1alpha1.DuplicateAction, tcParam, duration)
}

// SetCorrupt sets packet corruption on the network traffic
func (i *Instance) SetCorrupt(ctx context.Context, corrupt float32, duration time.Duration) error {
	tcParam := v1alpha1.TcParameter{
		Corrupt: &v1alpha1.CorruptSpec{
			Corrupt: fmt.Sprintf("%f", corrupt),
		},
	}
	return i.setNetworkChaos(ctx, v1alpha1.CorruptAction, tcParam, duration)
}

// SetBandwidth sets bandwidth limit on the network traffic
func (i *Instance) SetBandwidth(ctx context.Context, bandwidth *v1alpha1.BandwidthSpec, duration time.Duration) error {
	tcParam := v1alpha1.TcParameter{Bandwidth: bandwidth}
	return i.setNetworkChaos(ctx, v1alpha1.BandwidthAction, tcParam, duration)
}

// setNetworkChaos sets network chaos based on provided specifications
func (i *Instance) setNetworkChaos(ctx context.Context, action v1alpha1.NetworkChaosAction, tcParam v1alpha1.TcParameter, duration time.Duration) error {
	if i.gvr == (schema.GroupVersionResource{}) {
		return ErrChaosMeshNotEnabledInInstance.WithParams(i.Name())
	}

	if exists, err := i.CustomResourceDefinitionExists(ctx, &i.gvr); !exists {
		return ErrResourceMissing.Wrap(err)
	}

	durationStr := ptr.To(duration.String())
	if duration == 0 {
		durationStr = nil
	}

	netChaos := &v1alpha1.NetworkChaos{
		Spec: v1alpha1.NetworkChaosSpec{
			Action:      action,
			TcParameter: tcParam,
			Duration:    durationStr,
			Target: &v1alpha1.PodSelector{
				Selector: v1alpha1.PodSelectorSpec{},
				Mode:     v1alpha1.AllMode,
			},
			Direction: v1alpha1.Both,
			PodSelector: v1alpha1.PodSelector{
				Selector: v1alpha1.PodSelectorSpec{
					GenericSelectorSpec: v1alpha1.GenericSelectorSpec{
						LabelSelectors: i.Labels(),
					},
				},
				Mode: v1alpha1.AllMode,
			},
		},
	}

	netChaosObject := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": netChaos.ObjectMeta,
			"spec":     netChaos.Spec,
		},
	}

	if err := i.CreateCustomResource(ctx, &i.gvr, netChaosObject); err != nil {
		return ErrFailedToAddNetworkChaos.WithParams(action, i.Name()).Wrap(err)
	}

	injected, err := i.WaitForChaosInjection(ctx)
	if err != nil {
		return err
	}
	if !injected {
		return ErrFailedToInjectNetworkChaos.WithParams(action, i.Name())
	}

	return nil
}

// WaitForChaosInjection checks if the specified chaos actions are successfully applied
func (i *Instance) WaitForChaosInjection(ctx context.Context) (bool, error) {
	if i.gvr == (schema.GroupVersionResource{}) {
		return false, ErrChaosMeshNotEnabledInInstance.WithParams(i.k8sName)
	}

	for {
		injected, err := i.isChaosInjected(ctx)
		if err != nil {
			return false, err
		}

		if injected {
			return true, nil
		}

		select {
		case <-ctx.Done():
			return false, fmt.Errorf("timeout waiting for NetworkChaos %s to be fully injected: %w", i.k8sName, ctx.Err())
		case <-time.After(1 * time.Second):
			// Continue loop
		}
	}
}

// isChaosInjected checks if the chaos has been injected by examining the conditions
func (i *Instance) isChaosInjected(ctx context.Context) (bool, error) {
	netChaos, err := i.GetCustomResource(ctx, &i.gvr)
	if err != nil {
		return false, err
	}

	status, ok := netChaos.Object["status"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("failed to parse status of NetworkChaos %s", i.k8sName)
	}

	conditions, ok := status["conditions"].([]interface{})
	if !ok {
		return false, nil // Conditions not yet available, not an error, continue checking
	}

	for _, condition := range conditions {
		cond, ok := condition.(map[string]interface{})
		if !ok {
			continue
		}

		if cond["type"] == "AllInjected" && cond["status"] == "True" {
			return true, nil
		}
	}

	return false, nil
}

// IsChaosMeshAPIAvailable checks if the Chaos Mesh API is available on the cluster
func IsChaosMeshAPIAvailable(ctx context.Context, k8sClient k8s.KubeManager) (bool, error) {
	apiResourceList, err := k8sClient.Clientset().Discovery().
		ServerResourcesForGroupVersion(chaosMeshAPIGroupVersion)
	if err != nil {
		return false, err
	}

	requiredResources := []string{chaosMeshNetworkResource}
	for _, resource := range apiResourceList.APIResources {
		for i, requiredResource := range requiredResources {
			if resource.Name == requiredResource {
				requiredResources = append(requiredResources[:i], requiredResources[i+1:]...)
				break
			}
		}
	}

	if len(requiredResources) == 0 {
		return true, nil
	}
	return false, nil
}
