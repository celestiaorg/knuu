package k8s

import (
	"context"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateNetworkPolicy creates a new NetworkPolicy resource.
func CreateNetworkPolicy(
	namespace,
	name string,
	selectorMap,
	ingressSelectorMap,
	egressSelectorMap map[string]string,
) error {
	var ingress []v1.NetworkPolicyIngressRule
	if ingressSelectorMap != nil {
		ingress = []v1.NetworkPolicyIngressRule{
			{
				From: []v1.NetworkPolicyPeer{
					{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: ingressSelectorMap,
						},
					},
				},
			},
		}
	}

	var egress []v1.NetworkPolicyEgressRule
	if egressSelectorMap != nil {
		egress = []v1.NetworkPolicyEgressRule{
			{
				To: []v1.NetworkPolicyPeer{
					{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: egressSelectorMap,
						},
					},
				},
			},
		}
	}

	np := &v1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: v1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: selectorMap,
			},
			PolicyTypes: []v1.PolicyType{
				v1.PolicyTypeIngress,
				v1.PolicyTypeEgress,
			},
			Ingress: ingress,
			Egress:  egress,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !IsInitialized() {
		return ErrKnuuNotInitialized
	}
	_, err := Clientset().NetworkingV1().NetworkPolicies(namespace).Create(ctx, np, metav1.CreateOptions{})
	if err != nil {
		return ErrCreatingNetworkPolicy.WithParams(name).Wrap(err)
	}

	return nil
}

// DeleteNetworkPolicy removes a NetworkPolicy resource.
func DeleteNetworkPolicy(namespace string, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !IsInitialized() {
		return ErrKnuuNotInitialized
	}
	err := Clientset().NetworkingV1().NetworkPolicies(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return ErrDeletingNetworkPolicy.WithParams(name).Wrap(err)
	}

	return nil
}

// GetNetworkPolicy retrieves a NetworkPolicy resource.
func GetNetworkPolicy(namespace string, name string) (*v1.NetworkPolicy, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !IsInitialized() {
		return nil, ErrKnuuNotInitialized
	}
	np, err := Clientset().NetworkingV1().NetworkPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, ErrGettingNetworkPolicy.WithParams(name).Wrap(err)
	}

	return np, nil
}

// NetworkPolicyExists checks if a NetworkPolicy resource exists.
func NetworkPolicyExists(namespace string, name string) bool {
	_, err := GetNetworkPolicy(namespace, name)
	if err != nil {
		logrus.Debug("NetworkPolicy does not exist, err: ", err)
		return false
	}

	return true
}
