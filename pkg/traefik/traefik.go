package traefik

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/names"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	traefikServiceName = "traefik"
	Port               = 80
	PortSecure         = 443
	deploymentName     = "traefik-deployment"
	roleName           = "traefik-role"
	containerName      = "traefik"
	image              = "traefik:v3.0"
	appLabel           = "app"
	appLabelValue      = "traefik"
	replicas           = 2
	waitRetry          = 5 * time.Second
)

type Traefik struct {
	K8s      *k8s.Client
	endpoint string
}

func (t *Traefik) Deploy(ctx context.Context) error {
	if t.K8s == nil {
		return ErrTraefikClientNotInitialized
	}

	// Create a dedicated service account for Traefik
	serviceAccountName, err := names.NewRandomK8("traefik-service-account")
	if err != nil {
		return err
	}
	if err := t.K8s.CreateServiceAccount(ctx, serviceAccountName, nil); err != nil {
		return ErrFailedToCreateServiceAccount.Wrap(err)
	}

	clusterRoleName, err := names.NewRandomK8(roleName)
	if err != nil {
		return err
	}

	// Define and create a ClusterRole for Traefik
	err = t.K8s.CreateClusterRole(ctx, clusterRoleName, nil, []rbacv1.PolicyRule{
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

	if err := t.K8s.CreateClusterRoleBinding(ctx, clusterRoleName, nil, clusterRoleName, serviceAccountName); err != nil {
		return ErrTraefikRoleBindingCreationFailed.Wrap(err)
	}

	// Create the Traefik deployment using the service account
	traefikDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: t.K8s.Namespace(),
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
						},
					},
				},
			},
		},
	}
	_, err = t.K8s.Clientset().AppsV1().Deployments(t.K8s.Namespace()).Create(ctx, traefikDeployment, metav1.CreateOptions{})
	if err != nil {
		return ErrTraefikDeploymentCreationFailed.Wrap(err)
	}

	if err := t.K8s.WaitForDeployment(ctx, deploymentName); err != nil {
		return err
	}

	if err := t.createService(ctx); err != nil {
		return err
	}

	if err := t.K8s.WaitForService(ctx, traefikServiceName); err != nil {
		return err
	}

	return nil
}

func (t *Traefik) IP(ctx context.Context) (string, error) {
	if t.K8s == nil {
		return "", ErrTraefikClientNotInitialized
	}

	return t.K8s.GetServiceIP(ctx, traefikServiceName)
}

func (t *Traefik) URL(ctx context.Context, serviceName string) (string, error) {
	if t.endpoint == "" {
		var err error
		if t.endpoint, err = t.Endpoint(ctx); err != nil {
			return "", ErrTraefikIPNotFound.Wrap(err)
		}
	}
	return fmt.Sprintf("http://%s/%s", t.endpoint, serviceName), nil
}

func (t *Traefik) Endpoint(ctx context.Context) (string, error) {
	if t.K8s == nil {
		return "", ErrTraefikClientNotInitialized
	}
	return t.K8s.GetServiceEndpoint(ctx, traefikServiceName)
}

func (t *Traefik) AddHost(ctx context.Context, serviceName, prefix string, portsTCP ...int) error {
	middlewareName, err := names.NewRandomK8("strip-" + prefix)
	if err != nil {
		return ErrGeneratingRandomK8sName.Wrap(err)
	}

	// middleware is required to strip the prefix from the service name
	if err := t.createMiddleware(ctx, prefix, middlewareName); err != nil {
		return err
	}

	return t.createIngressRoute(ctx, serviceName, prefix, []string{middlewareName}, portsTCP)
}

// TODO: need to update the k8s pkg to handle service creation in more custom way
func (t *Traefik) createService(ctx context.Context) error {
	sCli := t.K8s.Clientset().CoreV1().Services(t.K8s.Namespace())

	srv := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      traefikServiceName,
			Namespace: t.K8s.Namespace(),
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

	logrus.Debugf("Service %s created successfully.", traefikServiceName)
	return nil
}

func (t *Traefik) createMiddleware(ctx context.Context, serviceName, middlewareName string) error {
	middleware := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "traefik.io/v1alpha1",
			"kind":       "Middleware",
			"metadata": map[string]interface{}{
				"name":      middlewareName,
				"namespace": t.K8s.Namespace(),
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

	_, err := t.K8s.DynamicClient().Resource(middlewareResource).Namespace(t.K8s.Namespace()).Create(ctx, middleware, metav1.CreateOptions{})
	if err != nil {
		return ErrTraefikMiddlewareCreationFailed.Wrap(err)
	}
	return nil
}

func (t *Traefik) createIngressRoute(
	ctx context.Context,
	serviceName, prefix string,
	middlewaresNames []string,
	ports []int,
) error {
	ingressRouteGVR := schema.GroupVersionResource{
		Group:    "traefik.io",
		Version:  "v1alpha1",
		Resource: "ingressroutes",
	}

	ingressRouteName, err := names.NewRandomK8("ing-route-" + prefix)
	if err != nil {
		return ErrTraefikIngressRouteCreationFailed.Wrap(err)
	}

	services := make([]interface{}, len(ports))
	for i, port := range ports {
		services[i] = map[string]interface{}{
			"name": serviceName,
			"port": port,
		}
	}

	middlewares := make([]interface{}, len(middlewaresNames))
	for i, name := range middlewaresNames {
		middlewares[i] = map[string]interface{}{
			"name": name,
		}
	}

	ingressRoute := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "traefik.io/v1alpha1",
			"kind":       "IngressRoute",
			"metadata": map[string]interface{}{
				"name":      ingressRouteName,
				"namespace": t.K8s.Namespace(),
			},
			"spec": map[string]interface{}{
				"entryPoints": []string{"web"},
				"routes": []interface{}{
					map[string]interface{}{
						"match":       fmt.Sprintf("PathPrefix(`/%s`)", prefix),
						"kind":        "Rule",
						"services":    services,
						"middlewares": middlewares,
					},
				},
			},
		},
	}

	_, err = t.K8s.DynamicClient().Resource(ingressRouteGVR).Namespace(t.K8s.Namespace()).Create(ctx, ingressRoute, metav1.CreateOptions{})
	if err != nil {
		return ErrTraefikIngressRouteCreationFailed.Wrap(err)
	}

	return nil
}
