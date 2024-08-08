package k8s

import (
	"context"

	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) CreateNetworkPolicy(
	ctx context.Context,
	name string,
	selectorMap,
	ingressSelectorMap,
	egressSelectorMap map[string]string,
) error {
	if err := validateNetworkPolicyName(name); err != nil {
		return err
	}
	if err := validateSelectorMap(selectorMap); err != nil {
		return err
	}
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
			Namespace: c.namespace,
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

	_, err := c.clientset.NetworkingV1().NetworkPolicies(c.namespace).Create(ctx, np, metav1.CreateOptions{})
	if err != nil {
		return ErrCreatingNetworkPolicy.WithParams(name).Wrap(err)
	}

	return nil
}

func (c *Client) DeleteNetworkPolicy(ctx context.Context, name string) error {
	err := c.clientset.NetworkingV1().NetworkPolicies(c.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return ErrDeletingNetworkPolicy.WithParams(name).Wrap(err)
	}

	return nil
}

func (c *Client) GetNetworkPolicy(ctx context.Context, name string) (*v1.NetworkPolicy, error) {
	np, err := c.clientset.NetworkingV1().NetworkPolicies(c.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, ErrGettingNetworkPolicy.WithParams(name).Wrap(err)
	}

	return np, nil
}

func (c *Client) NetworkPolicyExists(ctx context.Context, name string) bool {
	_, err := c.GetNetworkPolicy(ctx, name)
	if err != nil {
		c.logger.Debug("NetworkPolicy does not exist, err: ", err)
		return false
	}

	return true
}
