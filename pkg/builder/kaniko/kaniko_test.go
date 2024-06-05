package kaniko

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/k8s"
)

const (
	k8sNamespace    = "test-namespace"
	testImage       = "test-image"
	testDestination = "registry.example.com/test-image:latest"
)

func TestKanikoBuilder(t *testing.T) {
	k8sCS := fake.NewSimpleClientset()
	kb := &Kaniko{
		K8s: k8s.NewCustom(k8sCS, k8sCS.Discovery(), nil, k8sNamespace),
	}
	ctx := context.Background()

	t.Run("BuildSuccess", func(t *testing.T) {
		blCtx := "git://github.com/mojtaba-esk/sample-docker"
		cacheOpts := &builder.CacheOptions{}
		cacheOpts, err := cacheOpts.Default(blCtx)
		require.NoError(t, err, "GetDefaultCacheOptions should succeed")

		buildOptions := &builder.BuilderOptions{
			ImageName:    testImage,
			BuildContext: blCtx,
			Destination:  testDestination,
			Args:         []string{"--build-arg=value"},
			Cache:        cacheOpts,
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
		buildOptions := &builder.BuilderOptions{
			ImageName:    testImage,
			BuildContext: "git://example.com/repo",
			Destination:  testDestination,
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

func TestGetDefaultCacheOptions(t *testing.T) {
	t.Parallel()

	tt := []struct {
		buildContext  string
		expectedRepo  string
		expectedError bool
	}{
		{"git://example.com/repo", "ttl.sh/fd46c51aa5aff87d0f8a329fc578ffcb3b43f8db8aff920d0d01429e15eb9850:24h", false},
		{"", "", true},
	}

	for _, tc := range tt {
		t.Run(tc.buildContext, func(t *testing.T) {
			cacheOptions := &builder.CacheOptions{}
			cacheOptions, err := cacheOptions.Default(tc.buildContext)

			if tc.expectedError {
				assert.Error(t, err, "Expected an error, but got none")
				assert.Nil(t, cacheOptions, "Cache options should be nil on error")
			} else {
				assert.NoError(t, err, "Unexpected error")
				assert.NotNil(t, cacheOptions, "Cache options should not be nil")
				assert.Equal(t, tc.expectedRepo, cacheOptions.Repo, "Unexpected cache repo value")
			}
		})
	}
}
