package k8s

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applyv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/utils/ptr"
)

type ReplicaSetConfig struct {
	Name      string            // Name of the ReplicaSet
	Namespace string            // Namespace of the ReplicaSet
	Labels    map[string]string // Labels to apply to the ReplicaSet, key/value represents the name/value of the label
	Replicas  int32             // Replicas is the number of replicas
	PodConfig PodConfig         // PodConfig represents the pod configuration
}

// CreateReplicaSet creates a new replicaSet in namespace that k8s is initialized with if it doesn't already exist.
func (c *Client) CreateReplicaSet(ctx context.Context, rsConfig ReplicaSetConfig, init bool) (*appv1.ReplicaSet, error) {
	if c.terminated {
		return nil, ErrClientTerminated
	}
	if err := validateReplicaSetConfig(rsConfig); err != nil {
		return nil, err
	}
	rsConfig.Namespace = c.namespace
	newRs := c.prepareReplicaSet(rsConfig, init)

	existingRS, err := c.clientset.AppsV1().ReplicaSets(c.namespace).Get(ctx, rsConfig.Name, metav1.GetOptions{})
	if err == nil {
		newRs.Spec.Selector = applymetav1.LabelSelector().WithMatchLabels(existingRS.Spec.Selector.MatchLabels)
		newRs.Spec.Template.Labels = existingRS.Spec.Template.Labels
		newRs.Spec.Template.Annotations = existingRS.Spec.Template.Annotations

	} else if !errors.IsNotFound(err) {
		return nil, err
	}

	return c.clientset.AppsV1().ReplicaSets(c.namespace).
		Apply(ctx, newRs, metav1.ApplyOptions{FieldManager: fieldManager})
}

func (c *Client) ReplaceReplicaSetWithGracePeriod(ctx context.Context, ReplicaSetConfig ReplicaSetConfig, gracePeriod *int64) (*appv1.ReplicaSet, error) {
	c.logger.WithField("name", ReplicaSetConfig.Name).Debug("replacing replicaSet")

	if err := c.DeleteReplicaSetWithGracePeriod(ctx, ReplicaSetConfig.Name, gracePeriod); err != nil {
		return nil, ErrDeletingReplicaSet.Wrap(err)
	}

	if err := c.waitForReplicaSetDeletion(ctx, ReplicaSetConfig.Name); err != nil {
		return nil, ErrWaitingForReplicaSetDeletion.WithParams(ReplicaSetConfig.Name).Wrap(err)
	}

	createdRs, err := c.CreateReplicaSet(ctx, ReplicaSetConfig, false)
	if err != nil {
		return nil, ErrDeployingReplicaSet.Wrap(err)
	}

	return createdRs, nil
}

func (c *Client) ReplaceReplicaSet(ctx context.Context, ReplicaSetConfig ReplicaSetConfig) (*appv1.ReplicaSet, error) {
	return c.ReplaceReplicaSetWithGracePeriod(ctx, ReplicaSetConfig, nil)
}

func (c *Client) IsReplicaSetRunning(ctx context.Context, name string) (bool, error) {
	rs, err := c.getReplicaSet(ctx, name)
	if err != nil {
		return false, ErrGettingPod.WithParams(name).Wrap(err)
	}

	// Check if the ReplicaSet is running
	return rs.Status.ReadyReplicas == *rs.Spec.Replicas, nil
}

func (c *Client) DeleteReplicaSetWithGracePeriod(ctx context.Context, name string, gracePeriodSeconds *int64) error {
	exists, err := c.ReplicaSetExists(ctx, name)
	if err != nil {
		return ErrCheckingReplicaSetExists.WithParams(name).Wrap(err)
	}
	if !exists {
		return nil
	}
	if gracePeriodSeconds == nil {
		gracePeriodSeconds = ptr.To[int64](0)
	}

	delOpts := metav1.DeleteOptions{
		GracePeriodSeconds: gracePeriodSeconds,
	}
	if err := c.clientset.AppsV1().ReplicaSets(c.namespace).Delete(ctx, name, delOpts); err != nil {
		return ErrDeletingReplicaSet.WithParams(name).Wrap(err)
	}

	return nil
}

func (c *Client) DeleteReplicaSet(ctx context.Context, name string) error {
	return c.DeleteReplicaSetWithGracePeriod(ctx, name, nil)
}

func (c *Client) GetFirstPodFromReplicaSet(ctx context.Context, name string) (*v1.Pod, error) {
	rsName, err := c.getReplicaSet(ctx, name)
	if err != nil {
		// If the ReplicaSet does not exist, skip and return without error
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	selector := metav1.FormatLabelSelector(rsName.Spec.Selector)
	pods, err := c.clientset.CoreV1().Pods(c.namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, ErrListingPodsForReplicaSet.WithParams(name).Wrap(err)
	}

	if len(pods.Items) == 0 {
		return nil, ErrNoPodsForReplicaSet.WithParams(name)
	}

	return c.getPod(ctx, pods.Items[0].Name)
}

func (c *Client) getReplicaSet(ctx context.Context, name string) (*appv1.ReplicaSet, error) {
	if c.terminated {
		return nil, ErrClientTerminated
	}
	return c.clientset.AppsV1().ReplicaSets(c.namespace).Get(ctx, name, metav1.GetOptions{})
}

// ReplicaSetExists checks if a ReplicaSet exists in the namespace that k8s is initialized with.
func (c *Client) ReplicaSetExists(ctx context.Context, name string) (bool, error) {
	_, err := c.getReplicaSet(ctx, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, ErrGettingReplicaSet.WithParams(name).Wrap(err)
	}

	return true, nil
}

func (c *Client) waitForReplicaSetDeletion(ctx context.Context, name string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryInterval):
			exists, err := c.ReplicaSetExists(ctx, name)
			if err != nil {
				return ErrCheckingReplicaSetExists.WithParams(name).Wrap(err)
			}
			if !exists {
				// ReplicaSet has been deleted
				return nil
			}
		}
	}
}

// preparePod prepares a pod configuration.
func (c *Client) prepareReplicaSet(rsConf ReplicaSetConfig, init bool) *applyv1.ReplicaSetApplyConfiguration {
	rs := applyv1.ReplicaSet(rsConf.Name, rsConf.Namespace).
		WithLabels(rsConf.Labels).
		WithAPIVersion("apps/v1").
		WithKind("ReplicaSet").
		WithSpec(applyv1.ReplicaSetSpec().
			WithReplicas(rsConf.Replicas).
			WithSelector(applymetav1.LabelSelector().
				WithMatchLabels(rsConf.Labels),
			).
			WithTemplate(
				corev1.PodTemplateSpec().
					WithLabels(rsConf.Labels).
					WithAnnotations(rsConf.PodConfig.Annotations).
					WithSpec(c.preparePodSpec(rsConf.PodConfig, init)),
			),
		)

	c.logger.WithFields(logrus.Fields{
		"name":      rsConf.Name,
		"namespace": rsConf.Namespace,
	}).Debug("prepared replicaSet")
	return rs
}
