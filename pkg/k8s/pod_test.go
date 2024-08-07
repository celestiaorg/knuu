package k8s_test

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

func (s *TestSuite) TestDeployPod() {
	tests := []struct {
		name        string
		podConfig   k8s.PodConfig
		init        bool
		setupMock   func()
		expectedErr error
	}{
		{
			name: "successful creation",
			podConfig: k8s.PodConfig{
				Namespace:       s.namespace,
				Name:            "test-pod",
				Labels:          map[string]string{"app": "test"},
				ContainerConfig: testContainerConfig,
			},
			setupMock:   func() {},
			init:        false,
			expectedErr: nil,
		},
		{
			name: "client error",
			podConfig: k8s.PodConfig{
				Namespace:       s.namespace,
				Name:            "error-pod",
				Labels:          map[string]string{"app": "error"},
				ContainerConfig: testContainerConfig,
			},
			init: false,
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("create", "pods",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errInternalServerError
						})
			},
			expectedErr: k8s.ErrCreatingPod.Wrap(errInternalServerError),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			pod, err := s.client.DeployPod(context.Background(), tt.podConfig, tt.init)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
			s.Assert().Equal(tt.podConfig.Name, pod.Name)
		})
	}
}

func (s *TestSuite) TestReplacePod() {
	tests := []struct {
		name        string
		podConfig   k8s.PodConfig
		setupMock   func()
		expectedErr error
	}{
		{
			name: "successful replacement",
			podConfig: k8s.PodConfig{
				Namespace:       s.namespace,
				Name:            "test-pod",
				Labels:          map[string]string{"app": "test"},
				ContainerConfig: testContainerConfig,
			},
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "pods",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, nil
						})
			},
			expectedErr: nil,
		},
		{
			name: "client error on deletion",
			podConfig: k8s.PodConfig{
				Namespace:       s.namespace,
				Name:            "error-pod",
				Labels:          map[string]string{"app": "error"},
				ContainerConfig: testContainerConfig,
			},
			setupMock: func() {
				err := s.createPod("error-pod")
				s.Require().NoError(err)
				// The pod exist and there is some error deleting it.

				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "pods",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errInternalServerError
						})
			},
			expectedErr: k8s.ErrDeletingPod.Wrap(errInternalServerError),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			pod, err := s.client.ReplacePod(context.Background(), tt.podConfig)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
			s.Assert().Equal(tt.podConfig.Name, pod.Name)
		})
	}
}

func (s *TestSuite) TestIsPodRunning() {
	tests := []struct {
		name        string
		podName     string
		setupMock   func()
		expectedErr error
		expectedRun bool
	}{
		{
			name:    "pod is running",
			podName: "running-pod",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					CoreV1().Pods(s.namespace).
					Create(context.Background(), &v1.Pod{
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
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					CoreV1().Pods(s.namespace).
					Create(context.Background(), &v1.Pod{
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
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "pods",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errInternalServerError
						})
			},
			expectedRun: false,
			expectedErr: k8s.ErrGettingPod.WithParams("error-pod").Wrap(errInternalServerError),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			running, err := s.client.IsPodRunning(context.Background(), tt.podName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
			s.Assert().Equal(tt.expectedRun, running)
		})
	}
}

func (s *TestSuite) TestRunCommandInPod() {
	s.T().Skip("not implemented")
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

func (s *TestSuite) TestDeletePodWithGracePeriod() {
	tests := []struct {
		name        string
		podName     string
		gracePeriod *int64
		setupMock   func()
		expectedErr error
	}{
		{
			name:    "successful deletion",
			podName: "existing-pod",
			setupMock: func() {
				err := s.createPod("existing-pod")
				s.Require().NoError(err)
			},
			expectedErr: nil,
		},
		{
			name:        "pod not found",
			podName:     "non-existent-pod",
			setupMock:   func() {},
			expectedErr: nil,
		},
		{
			name:    "client error",
			podName: "error-pod",
			setupMock: func() {
				err := s.createPod("error-pod")
				s.Require().NoError(err)
				// The pod exist and there is some error deleting it.

				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "pods",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errInternalServerError
						})
			},
			expectedErr: k8s.ErrDeletingPodFailed.WithParams("error-pod").Wrap(errInternalServerError),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			err := s.client.DeletePodWithGracePeriod(context.Background(), tt.podName, tt.gracePeriod)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) TestDeletePod() {
	tests := []struct {
		name        string
		podName     string
		setupMock   func()
		expectedErr error
	}{
		{
			name:    "successful deletion",
			podName: "existing-pod",
			setupMock: func() {
				err := s.createPod("existing-pod")
				s.Require().NoError(err)
			},
			expectedErr: nil,
		},
		{
			name:        "pod not found",
			podName:     "non-existent-pod",
			setupMock:   func() {},
			expectedErr: nil,
		},
		{
			name:    "client error",
			podName: "error-pod",
			setupMock: func() {
				err := s.createPod("error-pod")
				s.Require().NoError(err)

				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "pods",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errInternalServerError
						})
			},
			expectedErr: k8s.ErrDeletingPodFailed.WithParams("error-pod").Wrap(errInternalServerError),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			err := s.client.DeletePod(context.Background(), tt.podName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) TestPortForwardPod() {
	s.T().Skip("not implemented")
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

func (s *TestSuite) createPod(name string) error {
	_, err := s.client.Clientset().CoreV1().Pods(s.namespace).Create(context.Background(), &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.namespace,
		},
	}, metav1.CreateOptions{})
	return err
}
