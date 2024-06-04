package k8s_test

import (
	"context"
	"errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

func (suite *TestSuite) TestDeployPod() {
	tests := []struct {
		name        string
		podConfig   k8s.PodConfig
		init        bool
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name: "successful creation",
			podConfig: k8s.PodConfig{
				Namespace: suite.namespace,
				Name:      "test-pod",
				Labels:    map[string]string{"app": "test"},
			},
			init:        false,
			setupMock:   func(clientset *fake.Clientset) {},
			expectedErr: nil,
		},
		{
			name: "client error",
			podConfig: k8s.PodConfig{
				Namespace: suite.namespace,
				Name:      "error-pod",
				Labels:    map[string]string{"app": "error"},
			},
			init: false,
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("create", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: k8s.ErrCreatingPod.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			pod, err := suite.client.DeployPod(context.Background(), tt.podConfig, tt.init)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), tt.podConfig.Name, pod.Name)
		})
	}
}

func (suite *TestSuite) TestReplacePod() {
	tests := []struct {
		name        string
		podConfig   k8s.PodConfig
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name: "successful replacement",
			podConfig: k8s.PodConfig{
				Namespace: suite.namespace,
				Name:      "test-pod",
				Labels:    map[string]string{"app": "test"},
			},
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("delete", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, nil
				})
			},
			expectedErr: nil,
		},
		{
			name: "client error on deletion",
			podConfig: k8s.PodConfig{
				Namespace: suite.namespace,
				Name:      "error-pod",
				Labels:    map[string]string{"app": "error"},
			},
			setupMock: func(clientset *fake.Clientset) {
				err := createPod(clientset, "error-pod", suite.namespace)
				require.NoError(suite.T(), err)
				// The pod exist and there is some error deleting it.

				clientset.PrependReactor("delete", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: k8s.ErrDeletingPod.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			pod, err := suite.client.ReplacePod(context.Background(), tt.podConfig)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), tt.podConfig.Name, pod.Name)
		})
	}
}

func (suite *TestSuite) TestIsPodRunning() {
	tests := []struct {
		name        string
		podName     string
		setupMock   func(*fake.Clientset)
		expectedErr error
		expectedRun bool
	}{
		{
			name:    "pod is running",
			podName: "running-pod",
			setupMock: func(clientset *fake.Clientset) {
				clientset.CoreV1().Pods(suite.namespace).Create(context.Background(), &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "running-pod",
					},
					Status: v1.PodStatus{
						ContainerStatuses: []v1.ContainerStatus{
							{
								Ready: true,
							},
						},
					},
				}, metav1.CreateOptions{})
			},
			expectedRun: true,
			expectedErr: nil,
		},
		{
			name:    "pod is not running",
			podName: "not-running-pod",
			setupMock: func(clientset *fake.Clientset) {
				clientset.CoreV1().Pods(suite.namespace).Create(context.Background(), &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "not-running-pod",
					},
					Status: v1.PodStatus{
						ContainerStatuses: []v1.ContainerStatus{
							{
								Ready: false,
							},
						},
					},
				}, metav1.CreateOptions{})
			},
			expectedRun: false,
			expectedErr: nil,
		},
		{
			name:    "client error",
			podName: "error-pod",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedRun: false,
			expectedErr: k8s.ErrGettingPod.WithParams("error-pod").Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			running, err := suite.client.IsPodRunning(context.Background(), tt.podName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), tt.expectedRun, running)
		})
	}
}

func (suite *TestSuite) TestRunCommandInPod() {
	suite.T().Skip("not implemented")
	// TestRunCommandInPod is not implemented.
	//
	// The RunCommandInPod function involves complex interactions with the Kubernetes API,
	// specifically around executing commands within a pod using SPDY protocol. This process
	// includes setting up SPDY streams and handling bi-directional communication, which are
	// challenging to accurately mock in a unit test environment.
	//
	// The primary reasons for not implementing a unit test for this function include:
	// 1. Dependency on SPDY protocol and complex networking interactions that are difficult to simulate.
	// 2. Requirement for real Kubernetes cluster behavior to validate the execution of commands within a pod.
	// 3. High complexity and low benefit of mocking deep Kubernetes internals and network protocols.
	//
	// Given these complexities, it is recommended to test the RunCommandInPod function in an
	// integration or end-to-end testing environment where real Kubernetes clusters and network
	// conditions can be used to validate its behavior.
}

func (suite *TestSuite) TestDeletePodWithGracePeriod() {
	tests := []struct {
		name        string
		podName     string
		gracePeriod *int64
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name:    "successful deletion",
			podName: "existing-pod",
			setupMock: func(clientset *fake.Clientset) {
				err := createPod(clientset, "existing-pod", suite.namespace)
				require.NoError(suite.T(), err)
			},
			expectedErr: nil,
		},
		{
			name:        "pod not found",
			podName:     "non-existent-pod",
			setupMock:   func(clientset *fake.Clientset) {},
			expectedErr: nil,
		},
		{
			name:    "client error",
			podName: "error-pod",
			setupMock: func(clientset *fake.Clientset) {
				err := createPod(clientset, "error-pod", suite.namespace)
				require.NoError(suite.T(), err)
				// The pod exist and there is some error deleting it.

				clientset.PrependReactor("delete", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: k8s.ErrDeletingPodFailed.WithParams("error-pod").Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			err := suite.client.DeletePodWithGracePeriod(context.Background(), tt.podName, tt.gracePeriod)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func (suite *TestSuite) TestDeletePod() {
	tests := []struct {
		name        string
		podName     string
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name:    "successful deletion",
			podName: "existing-pod",
			setupMock: func(clientset *fake.Clientset) {
				err := createPod(clientset, "existing-pod", suite.namespace)
				require.NoError(suite.T(), err)
			},
			expectedErr: nil,
		},
		{
			name:        "pod not found",
			podName:     "non-existent-pod",
			setupMock:   func(clientset *fake.Clientset) {},
			expectedErr: nil,
		},
		{
			name:    "client error",
			podName: "error-pod",
			setupMock: func(clientset *fake.Clientset) {
				err := createPod(clientset, "error-pod", suite.namespace)
				require.NoError(suite.T(), err)

				clientset.PrependReactor("delete", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: k8s.ErrDeletingPodFailed.WithParams("error-pod").Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			err := suite.client.DeletePod(context.Background(), tt.podName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func (suite *TestSuite) TestPortForwardPod() {
	suite.T().Skip("not implemented")
	// TestPortForwardPod is not implemented.
	//
	// The PortForwardPod function involves complex interactions with the Kubernetes API
	// that are difficult to mock accurately. Specifically, it relies on SPDY protocol
	// upgrades and bi-directional streaming, which are not easily simulated in a unit
	// testing environment.
	//
	// The primary challenges include:
	// - The use of spdy.RoundTripperFor to upgrade HTTP connections to SPDY, which
	//   involves lower-level network interactions that are not straightforward to
	//   mock with standard testing tools.
	// - The need to simulate bi-directional streaming between the local machine and
	//   the Kubernetes API server, which requires a robust networking setup.
	//
	// Given these complexities, it is recommended to test the PortForwardPod function
	// in an integration or end-to-end testing environment where real Kubernetes clusters
	// and network conditions can be used to validate its behavior.
}

func createPod(clientset *fake.Clientset, name, namespace string) error {
	_, err := clientset.CoreV1().Pods(namespace).Create(context.Background(), &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}, metav1.CreateOptions{})
	return err
}
