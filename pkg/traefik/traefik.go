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

	// // Define and create a role for Traefik
	// err = t.K8s.CreateRole(ctx, roleName, nil, []rbacv1.PolicyRule{
	// 	{
	// 		APIGroups: []string{""},
	// 		Resources: []string{"pods"},
	// 		Verbs:     []string{"get", "list", "watch"},
	// 	},
	// })

	// if err != nil {
	// 	return ErrTraefikRoleCreationFailed.Wrap(err)
	// }

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

func (t *Traefik) URL(ctx context.Context, name string) (string, error) {
	if t.endpoint == "" {
		var err error
		if t.endpoint, err = t.Endpoint(ctx); err != nil {
			return "", ErrTraefikIPNotFound.Wrap(err)
		}
	}
	return fmt.Sprintf("http://%s/%s", t.endpoint, name), nil
}

func (t *Traefik) Endpoint(ctx context.Context) (string, error) {
	if t.K8s == nil {
		return "", ErrTraefikClientNotInitialized
	}
	return t.K8s.GetServiceEndpoint(ctx, traefikServiceName)
}

func (t *Traefik) AddHost(ctx context.Context, serviceName string, port int) error {
	middleware := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "traefik.io/v1alpha1",
			"kind":       "Middleware",
			"metadata": map[string]interface{}{
				"name":      "strip-dummy",
				"namespace": "my-traefik-namespace",
			},
			"spec": map[string]interface{}{
				"stripPrefix": map[string]interface{}{
					"prefixes": []string{"/dummy"},
				},
			},
		},
	}

	middlewareResource := schema.GroupVersionResource{
		Group:    "traefik.io",
		Version:  "v1alpha1",
		Resource: "middlewares",
	}

	t.K8s.Clientset().

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
