package kaniko

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/system"
)

const (
	k8sNamespace    = "test-namespace"
	testImage       = "test-image"
	testDestination = "registry.example.com/test-image:latest"
)

func TestKanikoBuilder(t *testing.T) {
	k8sCS := fake.NewSimpleClientset()
	k8sClient, err := k8s.NewClientCustom(context.Background(), k8sCS, k8sCS.Discovery(), nil, k8sNamespace, logrus.New())
	require.NoError(t, err)
	kb, err := New(
		&system.SystemDependencies{
			K8sClient: k8sClient,
		},
		nil,
	)
	require.NoError(t, err)
	ctx := context.Background()

	t.Run("BuildSuccess", func(t *testing.T) {
		blCtx := "git://github.com/mojtaba-esk/sample-docker"
		buildOptions := builder.BuilderOptions{
			ImageName:    testImage,
			BuildContext: blCtx,
			Args:         []builder.ArgInterface{&builder.BuildArg{Value: "SOME_ARG=some_value"}},
			Cache:        kb.CacheOptions(),
		}

		var (
			logs string
			wg   = &sync.WaitGroup{}
		)
		wg.Add(1)
		go func() {
			defer wg.Done()
			logs, err = kb.Build(context.Background(), buildOptions)
		}()

		// Simulate the successful completion of the Job after a short delay
		time.Sleep(500 * time.Millisecond)
		completeAllJobInFakeClientset(t, k8sCS, k8sNamespace)

		wg.Wait()

		assert.NoError(t, err, "Build should succeed")
		assert.NotEmpty(t, logs, "Build logs should not be empty")
	})

	t.Run("BuildWithContextCancellation", func(t *testing.T) {
		buildOptions := builder.BuilderOptions{
			ImageName:    testImage,
			BuildContext: "git://example.com/repo",
		}

		// Cancel the context to simulate cancellation during the build
		ctx, cancel := context.WithCancel(ctx)
		cancel()

		logs, err := kb.Build(ctx, buildOptions)

		assert.Error(t, err, "build should fail due to context cancellation")
		assert.Empty(t, logs, "build logs should be empty")
	})

}

func completeAllJobInFakeClientset(t *testing.T, clientset *fake.Clientset, namespace string) {
	ctx := context.Background()

	job, err := clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	assert.NoError(t, err)

	for _, j := range job.Items {
		j.Status.Succeeded = 1
		_, err = clientset.BatchV1().Jobs(namespace).Update(ctx, &j, metav1.UpdateOptions{})
		require.NoError(t, err)

		// Create a Pod with the same name as the Job
		pod := createPodFromJob(&j)
		_, err = clientset.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
		require.NoError(t, err)
	}
}

func createPodFromJob(job *batchv1.Job) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job.Name,
			Namespace: job.Namespace,
			Labels: map[string]string{
				"job-name": job.Name,
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "fake-container",
					Image: "fake-image",
				},
			},
		},
	}
}
