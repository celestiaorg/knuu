package k8s

import (
	"context"
	"fmt"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateIngress creates a new Ingress resource.
func CreateIngress(namespace string, name string, labels map[string]string, annotations map[string]string, ingressClassName string, host string, path string, pathType string, serviceName string, servicePort int, tlsEnabled bool) error {

	var tls []v1.IngressTLS
	if tlsEnabled {
		tls = []v1.IngressTLS{
			{
				Hosts:      []string{host},
				SecretName: fmt.Sprintf("%s-tls", name),
			},
		}
	}

	pType := v1.PathType(pathType)

	rule := v1.IngressRule{
		Host: host,
		IngressRuleValue: v1.IngressRuleValue{
			HTTP: &v1.HTTPIngressRuleValue{
				Paths: []v1.HTTPIngressPath{
					{
						Path:     path,
						PathType: &pType,
						Backend: v1.IngressBackend{
							Service: &v1.IngressServiceBackend{
								Name: serviceName,
								Port: v1.ServiceBackendPort{
									Number: int32(servicePort),
								},
							},
						},
					},
				},
			},
		},
	}

	ingress := &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   namespace,
			Name:        name,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: v1.IngressSpec{
			IngressClassName: &ingressClassName,
			TLS:              tls,
			Rules: []v1.IngressRule{
				rule,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := Clientset().NetworkingV1().Ingresses(namespace).Create(ctx, ingress, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("creating Ingress resource: %w", err)
	}
	return nil
}

// DeleteIngress deletes an Ingress resource.
func DeleteIngress(namespace string, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := Clientset().NetworkingV1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("deleting Ingress resource: %w", err)
	}
	return nil
}
