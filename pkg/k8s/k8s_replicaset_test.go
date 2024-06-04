package k8s_test

import (
	"context"
	"errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/utils/ptr"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

func (suite *TestSuite) TestCreateReplicaSet() {
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
				Namespace: suite.namespace,
				Labels:    map[string]string{"app": "test"},
				Replicas:  1,
				PodConfig: k8s.PodConfig{
					Namespace: suite.namespace,
					Name:      "test-pod",
					Labels:    map[string]string{"app": "test"},
				},
			},
			init:        false,
			setupMock:   func() {},
			expectedErr: nil,
		},
		{
			name: "client error",
			rsConfig: k8s.ReplicaSetConfig{
				Name:      "error-rs",
				Namespace: suite.namespace,
				Labels:    map[string]string{"app": "error"},
				Replicas:  1,
				PodConfig: k8s.PodConfig{
					Namespace: suite.namespace,
					Name:      "error-pod",
					Labels:    map[string]string{"app": "error"},
				},
			},
			init: false,
			setupMock: func() {
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("create", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrCreatingReplicaSet.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock()

			rs, err := suite.client.CreateReplicaSet(context.Background(), tt.rsConfig, tt.init)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), tt.rsConfig.Name, rs.Name)
		})
	}
}

func (suite *TestSuite) TestReplaceReplicaSetWithGracePeriod() {
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
				Namespace: suite.namespace,
				Labels:    map[string]string{"app": "test"},
				Replicas:  1,
				PodConfig: k8s.PodConfig{
					Namespace: suite.namespace,
					Name:      "test-pod",
					Labels:    map[string]string{"app": "test"},
				},
			},
			gracePeriod: &gracePeriod,
			setupMock: func() {
				err := suite.createReplicaSet("test-rs")
				require.NoError(suite.T(), err)
			},
			expectedErr: nil,
		},
		{
			name: "client error on delete",
			rsConfig: k8s.ReplicaSetConfig{
				Name:      "error-rs",
				Namespace: suite.namespace,
				Labels:    map[string]string{"app": "error"},
				Replicas:  1,
				PodConfig: k8s.PodConfig{
					Namespace: suite.namespace,
					Name:      "error-pod",
					Labels:    map[string]string{"app": "error"},
				},
			},
			gracePeriod: &gracePeriod,
			setupMock: func() {
				// if it does not exist, it return nil as error
				// so we need to add it to the be bale to pass the existence check
				err := suite.createReplicaSet("error-rs")
				require.NoError(suite.T(), err)

				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrDeletingReplicaSet.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock()

			rs, err := suite.client.ReplaceReplicaSetWithGracePeriod(context.Background(), tt.rsConfig, tt.gracePeriod)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), tt.rsConfig.Name, rs.Name)
		})
	}
}

func (suite *TestSuite) TestReplaceReplicaSet() {
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
				Namespace: suite.namespace,
				Labels:    map[string]string{"app": "test"},
				Replicas:  1,
				PodConfig: k8s.PodConfig{
					Namespace: suite.namespace,
					Name:      "test-pod",
					Labels:    map[string]string{"app": "test"},
				},
			},
			setupMock: func() {
				err := suite.createReplicaSet("test-rs")
				require.NoError(suite.T(), err)
			},
			expectedErr: nil,
		},
		{
			name: "client error on delete",
			rsConfig: k8s.ReplicaSetConfig{
				Name:      "error-rs",
				Namespace: suite.namespace,
				Labels:    map[string]string{"app": "error"},
				Replicas:  1,
				PodConfig: k8s.PodConfig{
					Namespace: suite.namespace,
					Name:      "error-pod",
					Labels:    map[string]string{"app": "error"},
				},
			},
			setupMock: func() {
				// if it does not exist, it return nil as error
				// so we need to add it to the be bale to pass the existence check
				err := suite.createReplicaSet("error-rs")
				require.NoError(suite.T(), err)

				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrDeletingReplicaSet.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock()

			rs, err := suite.client.ReplaceReplicaSet(context.Background(), tt.rsConfig)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), tt.rsConfig.Name, rs.Name)
		})
	}
}

func (suite *TestSuite) TestIsReplicaSetRunning() {
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
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &appv1.ReplicaSet{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-rs",
									Namespace: suite.namespace,
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
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &appv1.ReplicaSet{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-rs",
									Namespace: suite.namespace,
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
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedRes: false,
			expectedErr: k8s.ErrGettingPod.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock()

			res, err := suite.client.IsReplicaSetRunning(context.Background(), tt.rsName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), tt.expectedRes, res)
		})
	}
}

func (suite *TestSuite) TestDeleteReplicaSetWithGracePeriod() {
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
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &appv1.ReplicaSet{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-rs",
									Namespace: suite.namespace,
								},
							}, nil
						})
				suite.client.Clientset().(*fake.Clientset).
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
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &appv1.ReplicaSet{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "error-rs",
									Namespace: suite.namespace,
								},
							}, nil
						})
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrDeletingReplicaSet.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock()

			err := suite.client.DeleteReplicaSetWithGracePeriod(context.Background(), tt.rsName, tt.gracePeriod)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func (suite *TestSuite) TestDeleteReplicaSet() {
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
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &appv1.ReplicaSet{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-rs",
									Namespace: suite.namespace,
								},
							}, nil
						})
				suite.client.Clientset().(*fake.Clientset).
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
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &appv1.ReplicaSet{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "error-rs",
									Namespace: suite.namespace,
								},
							}, nil
						})
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "replicasets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrDeletingReplicaSet.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock()

			err := suite.client.DeleteReplicaSet(context.Background(), tt.rsName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func (suite *TestSuite) createReplicaSet(name string) error {
	_, err := suite.client.Clientset().AppsV1().ReplicaSets(suite.namespace).Create(context.Background(), &appv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: suite.namespace,
		},
	}, metav1.CreateOptions{})
	return err
}
