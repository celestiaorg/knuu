package k8s

import (
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/celestiaorg/knuu/pkg/errors"
)

func validateDNS1123Label(name string, err *errors.Error) error {
	if errs := validation.IsDNS1123Label(name); len(errs) > 0 {
		return err.WithParams(name, errs)
	}
	return nil
}

func validateDNS1123Subdomain(name string, returnErr *errors.Error) error {
	if errs := validation.IsDNS1123Subdomain(name); len(errs) > 0 {
		return returnErr.WithParams(name, errs)
	}
	return nil
}

func ValidateNamespace(name string) error {
	return validateDNS1123Label(name, ErrInvalidNamespaceName)
}

func validateConfigMapName(name string) error {
	return validateDNS1123Subdomain(name, ErrInvalidConfigMapName)
}

func validateLabels(labels map[string]string) error {
	for key, value := range labels {
		if errs := validation.IsQualifiedName(key); len(errs) > 0 {
			return ErrInvalidLabelKey.WithParams(key, errs)
		}
		if errs := validation.IsValidLabelValue(value); len(errs) > 0 {
			return ErrInvalidLabelValue.WithParams(key, errs)
		}
	}
	return nil
}

func validateConfigMapKeys(data map[string]string) error {
	for key := range data {
		if errs := validation.IsConfigMapKey(key); len(errs) > 0 {
			return ErrInvalidConfigMapKey.WithParams(key, errs)
		}
	}
	return nil
}

func validateCustomResourceName(name string) error {
	return validateDNS1123Subdomain(name, ErrInvalidCustomResourceName)
}

func validateGroupVersionResource(gvr *schema.GroupVersionResource) error {
	if gvr.Group == "" || gvr.Version == "" || gvr.Resource == "" {
		return ErrInvalidGroupVersionResource.WithParams(gvr.Group, gvr.Version, gvr.Resource)
	}
	return nil
}

func validateCustomResourceObject(obj *map[string]interface{}) error {
	if obj == nil {
		return ErrCustomResourceObjectNil
	}
	if _, ok := (*obj)["spec"]; !ok {
		return ErrCustomResourceObjectNoSpec.WithParams(obj)
	}
	return nil
}

func validateDaemonSetName(name string) error {
	return validateDNS1123Subdomain(name, ErrInvalidDaemonSetName)
}

func validateContainers(containers []v1.Container) error {
	for _, container := range containers {
		if err := validateContainerName(container.Name); err != nil {
			return err
		}
	}
	return nil
}

func validateNetworkPolicyName(name string) error {
	return validateDNS1123Subdomain(name, ErrInvalidNetworkPolicyName)
}

func validateSelectorMap(selectorMap map[string]string) error {
	return validateLabels(selectorMap)
}

func validatePodName(name string) error {
	return validateDNS1123Subdomain(name, ErrInvalidPodName)
}

func validateContainerName(name string) error {
	return validateDNS1123Label(name, ErrInvalidContainerName)
}

func validateCommand(cmd []string) error {
	if len(cmd) == 0 {
		return ErrEmptyCommand
	}
	return nil
}

func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return ErrInvalidPort.WithParams(port)
	}
	return nil
}

func validatePodAnnotations(annotations map[string]string) error {
	for key, value := range annotations {
		if errs := validation.IsQualifiedName(key); len(errs) > 0 {
			return ErrInvalidPodAnnotationKey.WithParams(key, errs)
		}
		if len(value) > 256000 { // annotations have a maximum size of 256 KB
			return ErrAnnotationValueTooLarge.WithParams(key)
		}
	}
	return nil
}

func validateContainerConfig(config ContainerConfig) error {
	if len(config.Image) == 0 {
		return ErrContainerImageEmpty.WithParams(config.Name)
	}
	for _, volume := range config.Volumes {
		if err := validateVolume(volume); err != nil {
			return err
		}
	}
	for _, file := range config.Files {
		if err := validateFile(file); err != nil {
			return err
		}
	}
	return validateContainerName(config.Name)
}

func validateVolume(volume *Volume) error {
	if volume.Path == "" {
		return ErrVolumePathEmpty.WithParams(volume.Path)
	}

	if volume.Size.Value() <= 0 {
		return ErrVolumeSizeZero.WithParams(volume.Path)
	}
	return nil
}

func validateFile(file *File) error {
	if file.Source == "" || file.Dest == "" {
		return ErrFileSourceDestEmpty.WithParams(file.Source, file.Dest)
	}
	return nil
}

func validatePodConfig(podConfig PodConfig) error {
	if err := validatePodName(podConfig.Name); err != nil {
		return err
	}

	if err := ValidateNamespace(podConfig.Namespace); err != nil {
		return err
	}

	if err := validateLabels(podConfig.Labels); err != nil {
		return err
	}

	if err := validatePodAnnotations(podConfig.Annotations); err != nil {
		return err
	}

	if err := validateContainerConfig(podConfig.ContainerConfig); err != nil {
		return err
	}
	for _, sidecarConfig := range podConfig.SidecarConfigs {
		if err := validateContainerConfig(sidecarConfig); err != nil {
			return err
		}
	}

	return nil
}

func validatePVCName(name string) error {
	return validateDNS1123Subdomain(name, ErrInvalidPVCName)
}

func validatePVCSize(size resource.Quantity) error {
	if size.Value() <= 0 {
		return ErrPVCSizeZero.WithParams(size)
	}
	return nil
}

func validateReplicaSetName(name string) error {
	return validateDNS1123Subdomain(name, ErrInvalidReplicaSetName)
}

func validateReplicaSetConfig(rsConfig ReplicaSetConfig) error {
	if err := validateReplicaSetName(rsConfig.Name); err != nil {
		return err
	}
	if err := ValidateNamespace(rsConfig.Namespace); err != nil {
		return err
	}
	if err := validateLabels(rsConfig.Labels); err != nil {
		return err
	}
	if rsConfig.Replicas < 0 {
		return ErrReplicaSetReplicasNegative.WithParams(rsConfig.Replicas)
	}
	if err := validatePodConfig(rsConfig.PodConfig); err != nil {
		return err
	}
	return nil
}

func validateRoleName(name string) error {
	return validateDNS1123Subdomain(name, ErrInvalidRoleName)
}

func validatePolicyRules(policyRules []rbacv1.PolicyRule) error {
	for _, rule := range policyRules {
		if len(rule.Verbs) == 0 {
			return ErrPolicyRuleNoVerbs
		}
		for _, verb := range rule.Verbs {
			if len(verb) == 0 {
				return ErrPolicyRuleVerbEmpty
			}
		}
		if len(rule.Resources) == 0 && len(rule.NonResourceURLs) == 0 {
			return ErrPolicyRuleNoResources
		}
	}
	return nil
}

func validateClusterRoleName(name string) error {
	return validateDNS1123Subdomain(name, ErrInvalidClusterRoleName)
}

func validateRoleBindingName(name string) error {
	return validateDNS1123Subdomain(name, ErrInvalidRoleBindingName)
}

func validateServiceAccountName(name string) error {
	return validateDNS1123Subdomain(name, ErrInvalidServiceAccountName)
}

func validateClusterRoleBindingName(name string) error {
	return validateDNS1123Subdomain(name, ErrInvalidClusterRoleBindingName)
}

func validateServiceName(name string) error {
	return validateDNS1123Label(name, ErrInvalidServiceName)
}

func validatePorts(ports []int) error {
	for _, port := range ports {
		if err := validatePort(port); err != nil {
			return err
		}
	}
	return nil
}

func validateConfigMap(name string, labels, data map[string]string) error {
	if err := validateConfigMapName(name); err != nil {
		return err
	}
	if err := validateLabels(labels); err != nil {
		return err
	}
	return validateConfigMapKeys(data)
}

func validateServiceOptions(options ServiceOptions) error {
	if err := validateLabels(options.Labels); err != nil {
		return err
	}
	if err := validatePorts(options.TCPPorts); err != nil {
		return err
	}
	if err := validatePorts(options.UDPPorts); err != nil {
		return err
	}
	return validateSelectorMap(options.SelectorMap)
}
