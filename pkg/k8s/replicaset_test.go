package k8s_test

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	appv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/utils/ptr"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

func (s *TestSuite) TestCreateReplicaSet() {
	tests := []struct {
		name        string
		rsConfig    k8s.ReplicaSetConfig
		init        bool
		setupMock   func()
		expectedErr error
	}{
		{
			name: "successful creation",
			rsConfig: k8s.ReplicaSetConfig{
				Name:      "test-rs",
				Namespace: s.namespace,
				Labels:    map[string]string{"app": "test"},
				Replicas:  1,
				PodConfig: k8s.PodConfig{
					Namespace:       s.namespace,
					Name:            "test-pod",
					Labels:          map[string]string{"app": "test"},
					ContainerConfig: testContainerConfig,
				},
			},
			init: false,
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("patch", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							patchAction, ok := action.(k8stesting.PatchAction)
							if !ok {
								return false, nil, fmt.Errorf("expected PatchAction, got %T", action)
							}
							return true, &appsv1.ReplicaSet{
								ObjectMeta: metav1.ObjectMeta{
									Name:      patchAction.GetName(),
									Namespace: patchAction.GetNamespace(),
								},
							}, nil
						})
			},
			expectedErr: nil,
		},
		{
			name: "client error",
			rsConfig: k8s.ReplicaSetConfig{
				Name:      "error-rs",
				Namespace: s.namespace,
				Labels:    map[string]string{"app": "error"},
				Replicas:  1,
				PodConfig: k8s.PodConfig{
					Namespace:       s.namespace,
					Name:            "error-pod",
					Labels:          map[string]string{"app": "error"},
					ContainerConfig: testContainerConfig,
				},
			},
			init: false,
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					// we need to `patch` the replica set because the `apply` does not exist
					// and `apply` calls `patch` under the hood
					PrependReactor("patch", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errInternalServerError
						})
			},
			expectedErr: errInternalServerError,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			rs, err := s.client.CreateReplicaSet(context.Background(), tt.rsConfig, tt.init)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
			s.Assert().Equal(tt.rsConfig.Name, rs.Name)
		})
	}
}

func (s *TestSuite) TestReplaceReplicaSetWithGracePeriod() {
	gracePeriod := int64(10)
	tests := []struct {
		name        string
		rsConfig    k8s.ReplicaSetConfig
		gracePeriod *int64
		setupMock   func()
		expectedErr error
	}{
		{
			name: "successful replacement",
			rsConfig: k8s.ReplicaSetConfig{
				Name:      "test-rs",
				Namespace: s.namespace,
				Labels:    map[string]string{"app": "test"},
				Replicas:  1,
				PodConfig: k8s.PodConfig{
					Namespace:       s.namespace,
					Name:            "test-pod",
					Labels:          map[string]string{"app": "test"},
					ContainerConfig: testContainerConfig,
				},
			},
			gracePeriod: &gracePeriod,
			setupMock: func() {
				err := s.createReplicaSet("test-rs")
				s.Require().NoError(err)

				s.client.Clientset().(*fake.Clientset).
					PrependReactor("patch", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							patchAction, ok := action.(k8stesting.PatchAction)
							if !ok {
								return false, nil, fmt.Errorf("expected PatchAction, got %T", action)
							}
							return true, &appsv1.ReplicaSet{
								ObjectMeta: metav1.ObjectMeta{
									Name:      patchAction.GetName(),
									Namespace: patchAction.GetNamespace(),
								},
							}, nil
						})
			},
			expectedErr: nil,
		},
		{
			name: "client error on delete",
			rsConfig: k8s.ReplicaSetConfig{
				Name:      "error-rs",
				Namespace: s.namespace,
				Labels:    map[string]string{"app": "error"},
				Replicas:  1,
				PodConfig: k8s.PodConfig{
					Namespace:       s.namespace,
					Name:            "error-pod",
					Labels:          map[string]string{"app": "error"},
					ContainerConfig: testContainerConfig,
				},
			},
			gracePeriod: &gracePeriod,
			setupMock: func() {
				// if it does not exist, it return nil as error
				// so we need to add it to the be bale to pass the existence check
				err := s.createReplicaSet("error-rs")
				s.Require().NoError(err)

				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errInternalServerError
						})
			},
			expectedErr: k8s.ErrDeletingReplicaSet.Wrap(errInternalServerError),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			rs, err := s.client.ReplaceReplicaSetWithGracePeriod(context.Background(), tt.rsConfig, tt.gracePeriod)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
			s.Assert().Equal(tt.rsConfig.Name, rs.Name)
		})
	}
}

func (s *TestSuite) TestReplaceReplicaSet() {
	tests := []struct {
		name        string
		rsConfig    k8s.ReplicaSetConfig
		setupMock   func()
		expectedErr error
	}{
		{
			name: "successful replacement",
			rsConfig: k8s.ReplicaSetConfig{
				Name:      "test-rs",
				Namespace: s.namespace,
				Labels:    map[string]string{"app": "test"},
				Replicas:  1,
				PodConfig: k8s.PodConfig{
					Namespace:       s.namespace,
					Name:            "test-pod",
					Labels:          map[string]string{"app": "test"},
					ContainerConfig: testContainerConfig,
				},
			},
			setupMock: func() {
				err := s.createReplicaSet("test-rs")
				s.Require().NoError(err)

				s.client.Clientset().(*fake.Clientset).
					PrependReactor("patch", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							patchAction, ok := action.(k8stesting.PatchAction)
							if !ok {
								return false, nil, fmt.Errorf("expected PatchAction, got %T", action)
							}
							return true, &appsv1.ReplicaSet{
								ObjectMeta: metav1.ObjectMeta{
									Name:      patchAction.GetName(),
									Namespace: patchAction.GetNamespace(),
								},
							}, nil
						})
			},
			expectedErr: nil,
		},
		{
			name: "client error on delete",
			rsConfig: k8s.ReplicaSetConfig{
				Name:      "error-rs",
				Namespace: s.namespace,
				Labels:    map[string]string{"app": "error"},
				Replicas:  1,
				PodConfig: k8s.PodConfig{
					Namespace:       s.namespace,
					Name:            "error-pod",
					Labels:          map[string]string{"app": "error"},
					ContainerConfig: testContainerConfig,
				},
			},
			setupMock: func() {
				// if it does not exist, it return nil as error
				// so we need to add it to the be bale to pass the existence check
				err := s.createReplicaSet("error-rs")
				s.Require().NoError(err)

				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errInternalServerError
						})
			},
			expectedErr: k8s.ErrDeletingReplicaSet.Wrap(errInternalServerError),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			rs, err := s.client.ReplaceReplicaSet(context.Background(), tt.rsConfig)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
			s.Assert().Equal(tt.rsConfig.Name, rs.Name)
		})
	}
}

func (s *TestSuite) TestIsReplicaSetRunning() {
	tests := []struct {
		name        string
		rsName      string
		setupMock   func()
		expectedRes bool
		expectedErr error
	}{
		{
			name:   "replica set is running",
			rsName: "test-rs",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &appv1.ReplicaSet{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-rs",
									Namespace: s.namespace,
								},
								Spec: appv1.ReplicaSetSpec{
									Replicas: ptr.To[int32](1),
								},
								Status: appv1.ReplicaSetStatus{
									ReadyReplicas: 1,
								},
							}, nil
						})
			},
			expectedRes: true,
			expectedErr: nil,
		},
		{
			name:   "replica set is not running",
			rsName: "test-rs",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &appv1.ReplicaSet{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-rs",
									Namespace: s.namespace,
								},
								Spec: appv1.ReplicaSetSpec{
									Replicas: ptr.To[int32](1),
								},
								Status: appv1.ReplicaSetStatus{
									ReadyReplicas: 0,
								},
							}, nil
						})
			},
			expectedRes: false,
			expectedErr: nil,
		},
		{
			name:   "client error",
			rsName: "error-rs",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errInternalServerError
						})
			},
			expectedRes: false,
			expectedErr: k8s.ErrGettingPod.Wrap(errInternalServerError),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			res, err := s.client.IsReplicaSetRunning(context.Background(), tt.rsName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
			s.Assert().Equal(tt.expectedRes, res)
		})
	}
}

func (s *TestSuite) TestDeleteReplicaSetWithGracePeriod() {
	gracePeriod := int64(10)
	tests := []struct {
		name        string
		rsName      string
		gracePeriod *int64
		setupMock   func()
		expectedErr error
	}{
		{
			name:        "successful deletion",
			rsName:      "test-rs",
			gracePeriod: &gracePeriod,
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &appv1.ReplicaSet{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-rs",
									Namespace: s.namespace,
								},
							}, nil
						})
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, nil
						})
			},
			expectedErr: nil,
		},
		{
			name:        "replica set not found",
			rsName:      "missing-rs",
			gracePeriod: &gracePeriod,
			setupMock:   func() {},
			expectedErr: nil,
		},
		{
			name:        "client error on delete",
			rsName:      "error-rs",
			gracePeriod: &gracePeriod,
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &appv1.ReplicaSet{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "error-rs",
									Namespace: s.namespace,
								},
							}, nil
						})
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errInternalServerError
						})
			},
			expectedErr: k8s.ErrDeletingReplicaSet.Wrap(errInternalServerError),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			err := s.client.DeleteReplicaSetWithGracePeriod(context.Background(), tt.rsName, tt.gracePeriod)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) TestDeleteReplicaSet() {
	tests := []struct {
		name        string
		rsName      string
		setupMock   func()
		expectedErr error
	}{
		{
			name:   "successful deletion",
			rsName: "test-rs",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &appv1.ReplicaSet{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-rs",
									Namespace: s.namespace,
								},
							}, nil
						})
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, nil
						})
			},
			expectedErr: nil,
		},
		{
			name:        "replica set not found",
			rsName:      "missing-rs",
			setupMock:   func() {},
			expectedErr: nil,
		},
		{
			name:   "client error on delete",
			rsName: "error-rs",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &appv1.ReplicaSet{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "error-rs",
									Namespace: s.namespace,
								},
							}, nil
						})
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errInternalServerError
						})
			},
			expectedErr: k8s.ErrDeletingReplicaSet.Wrap(errInternalServerError),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			err := s.client.DeleteReplicaSet(context.Background(), tt.rsName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) createReplicaSet(name string) error {
	_, err := s.client.Clientset().AppsV1().ReplicaSets(s.namespace).Create(context.Background(), &appv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.namespace,
		},
	}, metav1.CreateOptions{})
	return err
}
