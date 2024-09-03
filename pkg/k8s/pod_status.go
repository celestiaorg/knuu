package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodStatus struct {
	Name            string
	Status          corev1.PodPhase
	PendingDuration time.Duration
}

// AllPodsStatuses reports the status of pods in the current namespace.
func (c *Client) AllPodsStatuses(ctx context.Context) ([]PodStatus, error) {
	pods, err := c.clientset.CoreV1().
		Pods(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, ErrListingPods.Wrap(err)
	}

	if len(pods.Items) == 0 {
		return nil, nil
	}

	output := make([]PodStatus, 0, len(pods.Items))
	for _, pod := range pods.Items {
		pendingDuration := time.Duration(0)
		if pod.Status.Phase == corev1.PodPending {
			pendingDuration = time.Since(pod.CreationTimestamp.Time)
		}

		output = append(output, PodStatus{
			Name:            pod.Name,
			Status:          pod.Status.Phase,
			PendingDuration: pendingDuration,
		})
	}
	return output, nil
}

func (c *Client) PodStatus(ctx context.Context, name string) (PodStatus, error) {
	pod, err := c.clientset.CoreV1().
		Pods(c.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return PodStatus{}, ErrGetPodStatus.WithParams(name).Wrap(err)
	}

	pendingDuration := time.Duration(0)
	if pod.Status.Phase == corev1.PodPending {
		pendingDuration = time.Since(pod.CreationTimestamp.Time)
	}

	return PodStatus{
		Name:            pod.Name,
		Status:          pod.Status.Phase,
		PendingDuration: pendingDuration,
	}, nil
}

func (c *Client) PrintAllPodsStatuses(ctx context.Context) error {
	statuses, err := c.AllPodsStatuses(ctx)
	if err != nil {
		return err
	}

	for _, s := range statuses {
		fmt.Printf("%-60s | %s\n", s.Name, s.Status)
	}
	return nil
}

// reportLongPendingPods checks for pods that have been pending longer than the specified maxPendingDuration
// and logs a warning message with the pods that are pending for too long.
func (c *Client) reportLongPendingPods(ctx context.Context) error {
	statuses, err := c.AllPodsStatuses(ctx)
	if err != nil {
		return err
	}

	// Collect pods that have been pending longer than the allowed duration
	longPendingPods := make([]string, 0)
	for _, s := range statuses {
		if s.Status == corev1.PodPending && s.PendingDuration > c.maxPendingDuration {
			longPendingPods = append(longPendingPods, s.Name)
		}
	}

	// If there are no pods pending too long, return nil
	if len(longPendingPods) == 0 {
		return nil
	}

	c.logger.WithField("pending_pods", strings.Join(longPendingPods, ", ")).Warn("Pods pending for too long")
	c.logger.WithField("pod_statuses", generatePodsStatusSummary(statuses)).Info("Pod statuses")
	return nil
}

// startPendingPodsWarningMonitor starts a background process to periodically check for pods that have been pending longer than the maxPendingDuration.
func (c *Client) startPendingPodsWarningMonitor(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(c.maxPendingDuration)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := c.reportLongPendingPods(ctx); err != nil {
					c.logger.WithError(err).Error("failed to report long pending pods")
				}
			case <-ctx.Done():
				c.logger.Infof("Shutting down long pending pods monitor.")
				return
			}
		}
	}()
}

func generatePodsStatusSummary(statuses []PodStatus) string {
	summary := make(map[corev1.PodPhase]int)
	for _, s := range statuses {
		summary[s.Status]++
	}

	output := ""
	for status, count := range summary {
		output += fmt.Sprintf("%s: %d , ", status, count)
	}
	return strings.TrimSuffix(output, ", ")
}
