package traefik

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

const (
	traefikServiceName     = "traefik"
	traefikAPIGroupVersion = "traefik.io/v1alpha1"
	Port                   = 80
	PortSecure             = 443
	deploymentName         = "traefik-deployment"
	serviceAccountName     = "traefik-service-account"
	roleName               = "traefik-role"
	containerName          = "traefik"
	image                  = "traefik:v3.0"
	appLabel               = "app"
	appLabelValue          = "traefik"
	replicas               = 1
	waitRetry              = 5 * time.Second

	defaultCPURequest    = "500m"
	defaultMemoryRequest = "500Mi"
	maxCPULimit          = "1000m"
	maxMemoryLimit       = "750Mi"
)

type Traefik struct {
	K8sClient k8s.KubeManager
	Logger    *logrus.Logger
	endpoint  string
}

func (t *Traefik) Deploy(ctx context.Context) error {
	if t.K8sClient == nil {
		return ErrTraefikClientNotInitialized
	}

	if err := t.K8sClient.CreateServiceAccount(ctx, serviceAccountName, nil); err != nil {
		if !errors.Is(err, k8s.ErrServiceAccountAlreadyExists) {
			return ErrFailedToCreateServiceAccount.Wrap(err)
		}
	}

	clusterRoleName := k8s.SanitizeName(t.K8sClient.Namespace() + "-" + roleName)

	// Define and create a ClusterRole for Traefik
	err := t.K8sClient.CreateClusterRole(ctx, clusterRoleName, nil, []rbacv1.PolicyRule{
		{
			APIGroups: []string{""}, // Core group
			Resources: []string{"pods", "endpoints", "secrets", "services"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"traefik.io"}, // Traefik specific resources
			Resources: []string{
				"ingressroutes", "middlewares", "tlsstores", "serverstransporttcps",
				"traefikservices", "ingressrouteudps", "middlewaretcps", "tlsoptions",
				"serverstransports", "ingressroutetcps",
			},
			Verbs: []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"networking.k8s.io"}, // Networking resources
			Resources: []string{"ingresses", "ingressclasses"},
			Verbs:     []string{"get", "list", "watch"},
		},
	})

	if err != nil {
		return ErrTraefikRoleCreationFailed.Wrap(err)
	}

	if err := t.K8sClient.CreateClusterRoleBinding(ctx, clusterRoleName, nil, clusterRoleName, serviceAccountName); err != nil {
		return ErrTraefikRoleBindingCreationFailed.Wrap(err)
	}

	cpuReq, err := resource.ParseQuantity(defaultCPURequest)
	if err != nil {
		return ErrTraefikFailedToParseQuantity.Wrap(err)
	}
	memReq, err := resource.ParseQuantity(defaultMemoryRequest)
	if err != nil {
		return ErrTraefikFailedToParseQuantity.Wrap(err)
	}
	cpuLimit, err := resource.ParseQuantity(maxCPULimit)
	if err != nil {
		return ErrTraefikFailedToParseQuantity.Wrap(err)
	}
	memLimit, err := resource.ParseQuantity(maxMemoryLimit)
	if err != nil {
		return ErrTraefikFailedToParseQuantity.Wrap(err)
	}

	// Create the Traefik deployment using the service account
	traefikDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: t.K8sClient.Namespace(),
			Labels:    map[string]string{appLabel: appLabelValue},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To[int32](replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{appLabel: appLabelValue},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{appLabel: appLabelValue},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: serviceAccountName,
					Containers: []v1.Container{
						{
							Name:  containerName,
							Image: image,
							Args: []string{
								"--api.insecure=true",
								"--providers.kubernetesIngress",
								"--providers.kubernetesCRD",
								fmt.Sprintf("--entrypoints.web.Address=:%d", Port),
								fmt.Sprintf("--entrypoints.websecure.Address=:%d", PortSecure),
							},
							Ports: []v1.ContainerPort{
								{ContainerPort: Port, Name: "web"},
								{ContainerPort: PortSecure, Name: "websecure"},
							},
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    cpuReq,
									v1.ResourceMemory: memReq,
								},
								Limits: v1.ResourceList{
									v1.ResourceCPU:    cpuLimit,
									v1.ResourceMemory: memLimit,
								},
							},
						},
					},
				},
			},
		},
	}
	_, err = t.K8sClient.Clientset().AppsV1().Deployments(t.K8sClient.Namespace()).
		Create(ctx, traefikDeployment, metav1.CreateOptions{})
	if err != nil {
		return ErrTraefikDeploymentCreationFailed.Wrap(err)
	}

	if err := t.K8sClient.WaitForDeployment(ctx, deploymentName); err != nil {
		return err
	}

	if err := t.createService(ctx); err != nil {
		return err
	}

	if err := t.K8sClient.WaitForService(ctx, traefikServiceName); err != nil {
		return err
	}

	return nil
}

func (t *Traefik) IP(ctx context.Context) (string, error) {
	if t.K8sClient == nil {
		return "", ErrTraefikClientNotInitialized
	}

	return t.K8sClient.GetServiceIP(ctx, traefikServiceName)
}

func (t *Traefik) URL(ctx context.Context, prefix string) (string, error) {
	if t.endpoint == "" {
		var err error
		if t.endpoint, err = t.Endpoint(ctx); err != nil {
			return "", ErrTraefikIPNotFound.Wrap(err)
		}
	}
	return fmt.Sprintf("http://%s/%s", t.endpoint, prefix), nil
}

func (t *Traefik) Endpoint(ctx context.Context) (string, error) {
	if t.K8sClient == nil {
		return "", ErrTraefikClientNotInitialized
	}
	return t.K8sClient.GetServiceEndpoint(ctx, traefikServiceName)
}

func (t *Traefik) AddHost(ctx context.Context, serviceName, prefix string, portTCP int) error {
	middlewareName := k8s.SanitizeName(prefix + "-strip")

	// middleware is required to strip the prefix from the service name
	if err := t.createMiddleware(ctx, prefix, middlewareName); err != nil {
		return err
	}

	return t.createIngressRoute(ctx, serviceName, prefix, middlewareName, portTCP)
}

// TODO: need to update the k8s pkg to handle service creation in more custom way
func (t *Traefik) createService(ctx context.Context) error {
	sCli := t.K8sClient.Clientset().CoreV1().Services(t.K8sClient.Namespace())

	srv := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      traefikServiceName,
			Namespace: t.K8sClient.Namespace(),
			Labels:    map[string]string{appLabel: appLabelValue},
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{appLabel: appLabelValue},
			Ports: []v1.ServicePort{
				{
					Name:       "web",
					Protocol:   v1.ProtocolTCP,
					Port:       Port,
					TargetPort: intstr.FromInt(Port),
				},
				{
					Name:       "websecure",
					Protocol:   v1.ProtocolTCP,
					Port:       PortSecure,
					TargetPort: intstr.FromInt(PortSecure),
				},
			},
			Type: v1.ServiceTypeLoadBalancer,
		},
	}

	if _, err := sCli.Create(ctx, srv, metav1.CreateOptions{}); err != nil {
		return ErrTraefikFailedToCreateService.Wrap(err)
	}

	t.Logger.WithField("service", traefikServiceName).Debug("Service created successfully.")
	return nil
}

func (t *Traefik) createMiddleware(ctx context.Context, serviceName, middlewareName string) error {
	middleware := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "traefik.io/v1alpha1",
			"kind":       "Middleware",
			"metadata": map[string]interface{}{
				"name":      middlewareName,
				"namespace": t.K8sClient.Namespace(),
			},
			"spec": map[string]interface{}{
				"stripPrefix": map[string]interface{}{
					"prefixes": []string{"/" + serviceName},
				},
			},
		},
	}

	middlewareResource := schema.GroupVersionResource{
		Group:    "traefik.io",
		Version:  "v1alpha1",
		Resource: "middlewares",
	}

	_, err := t.K8sClient.DynamicClient().Resource(middlewareResource).Namespace(t.K8sClient.Namespace()).
		Create(ctx, middleware, metav1.CreateOptions{})
	if err != nil {
		return ErrTraefikMiddlewareCreationFailed.Wrap(err)
	}
	return nil
}

func (t *Traefik) createIngressRoute(
	ctx context.Context,
	serviceName, prefix string,
	middlewareName string,
	port int,
) error {
	ingressRouteGVR := schema.GroupVersionResource{
		Group:    "traefik.io",
		Version:  "v1alpha1",
		Resource: "ingressroutes",
	}

	ingressRouteName := k8s.SanitizeName(prefix + "-ing-route")
	ingressRoute := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "traefik.io/v1alpha1",
			"kind":       "IngressRoute",
			"metadata": map[string]interface{}{
				"name":      ingressRouteName,
				"namespace": t.K8sClient.Namespace(),
			},
			"spec": map[string]interface{}{
				"entryPoints": []string{"web"},
				"routes": []interface{}{
					map[string]interface{}{
						"match": fmt.Sprintf("PathPrefix(`/%s`)", prefix),
						"kind":  "Rule",
						"services": []interface{}{
							map[string]interface{}{
								"name":      serviceName,
								"namespace": t.K8sClient.Namespace(),
								"port":      port,
							},
						},
						"middlewares": []interface{}{
							map[string]interface{}{
								"name":      middlewareName,
								"namespace": t.K8sClient.Namespace(),
							},
						},
					},
				},
			},
		},
	}

	_, err := t.K8sClient.DynamicClient().Resource(ingressRouteGVR).Namespace(t.K8sClient.Namespace()).
		Create(ctx, ingressRoute, metav1.CreateOptions{})
	if err != nil {
		return ErrTraefikIngressRouteCreationFailed.Wrap(err)
	}

	return nil
}

// IsTraefikAPIAvailable checks if the Traefik API is available in the cluster.
func (t *Traefik) IsTraefikAPIAvailable(ctx context.Context) bool {
	apiResourceList, err := t.K8sClient.Clientset().Discovery().ServerResourcesForGroupVersion(traefikAPIGroupVersion)
	if err != nil {
		t.Logger.WithError(err).Error("Failed to discover Traefik API resources")
		return false
	}

	requiredResources := []string{"middlewares", "ingressroutes"}

	for _, resource := range apiResourceList.APIResources {
		for i, requiredResource := range requiredResources {
			if resource.Name == requiredResource {
				requiredResources = append(requiredResources[:i], requiredResources[i+1:]...)
				break
			}
		}
	}

	if len(requiredResources) == 0 {
		return true
	}

	t.Logger.WithField("missing_resources", requiredResources).Warn("Missing Traefik API resources")
	return false
}
