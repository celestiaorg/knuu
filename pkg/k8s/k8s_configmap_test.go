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

func (suite *TestSuite) TestGetConfigMap() {
	tests := []struct {
		name          string
		configMapName string
		setupMock     func(*fake.Clientset)
		expectedErr   error
		expectedCM    *v1.ConfigMap
	}{
		{
			name:          "successful retrieval",
			configMapName: "test-configmap",
			setupMock: func(clientset *fake.Clientset) {
				err := createConfigMap(clientset, "test-configmap", suite.namespace)
				require.NoError(suite.T(), err)
			},
			expectedErr: nil,
			expectedCM: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-configmap",
					Namespace: suite.namespace,
				},
			},
		},
		{
			name:          "configmap not found",
			configMapName: "non-existent-configmap",
			setupMock: func(clientset *fake.Clientset) {
				// No setup needed for this case
			},
			expectedErr: k8s.ErrGettingConfigmap.WithParams("non-existent-configmap").
				Wrap(errors.New("configmaps \"non-existent-configmap\" not found")),
			expectedCM: nil,
		},
		{
			name:          "client error",
			configMapName: "error-configmap",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "configmaps", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: k8s.ErrGettingConfigmap.WithParams("error-configmap").
				Wrap(errors.New("internal server error")),
			expectedCM: nil,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			cm, err := suite.client.GetConfigMap(context.Background(), tt.configMapName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
			assert.EqualValues(suite.T(), tt.expectedCM, cm)
		})
	}
}

func (suite *TestSuite) TestConfigMapExists() {
	tests := []struct {
		name          string
		configMapName string
		setupMock     func(*fake.Clientset)
		expectedExist bool
		expectedErr   error
	}{
		{
			name:          "configmap exists",
			configMapName: "existing-configmap",
			setupMock: func(clientset *fake.Clientset) {
				err := createConfigMap(clientset, "existing-configmap", suite.namespace)
				require.NoError(suite.T(), err)
			},
			expectedExist: true,
			expectedErr:   nil,
		},
		{
			name:          "configmap does not exist",
			configMapName: "non-existent-configmap",
			setupMock:     func(clientset *fake.Clientset) {},
			expectedExist: false,
			expectedErr:   nil,
		},
		{
			name:          "client error",
			configMapName: "error-configmap",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "configmaps", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedExist: false,
			expectedErr: k8s.ErrGettingConfigmap.WithParams("error-configmap").
				Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			exists, err := suite.client.ConfigMapExists(context.Background(), tt.configMapName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), tt.expectedExist, exists)
		})
	}
}

func (suite *TestSuite) TestCreateConfigMap() {
	tests := []struct {
		name        string
		configMap   *v1.ConfigMap
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name: "successful creation",
			configMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-configmap",
					Namespace: suite.namespace,
				},
				Data: map[string]string{"key": "value"},
			},
			setupMock:   func(clientset *fake.Clientset) {},
			expectedErr: nil,
		},
		{
			name: "configmap already exists",
			configMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-configmap",
					Namespace: suite.namespace,
				},
			},
			setupMock: func(clientset *fake.Clientset) {
				err := createConfigMap(clientset, "existing-configmap", suite.namespace)
				require.NoError(suite.T(), err)
			},
			expectedErr: k8s.ErrConfigmapAlreadyExists.WithParams("existing-configmap").
				Wrap(errors.New("configmap already exists")),
		},
		{
			name: "client error",
			configMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "error-configmap",
					Namespace: suite.namespace,
				},
			},
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("create", "configmaps", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: k8s.ErrCreatingConfigmap.WithParams("error-configmap").
				Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			cm, err := suite.client.CreateConfigMap(context.Background(), tt.configMap.Name, tt.configMap.Labels, tt.configMap.Data)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
			assert.EqualValues(suite.T(), tt.configMap, cm)
		})
	}
}

func (suite *TestSuite) TestDeleteConfigMap() {
	tests := []struct {
		name          string
		configMapName string
		setupMock     func(*fake.Clientset)
		expectedErr   error
	}{
		{
			name:          "successful deletion",
			configMapName: "existing-configmap",
			setupMock: func(clientset *fake.Clientset) {
				err := createConfigMap(clientset, "existing-configmap", suite.namespace)
				require.NoError(suite.T(), err)
			},
			expectedErr: nil,
		},
		{
			name:          "configmap does not exist",
			configMapName: "non-existent-configmap",
			setupMock:     func(clientset *fake.Clientset) {},
			expectedErr: k8s.ErrConfigmapDoesNotExist.WithParams("non-existent-configmap").
				Wrap(errors.New("configmap does not exist")),
		},
		{
			name:          "client error",
			configMapName: "error-configmap",
			setupMock: func(clientset *fake.Clientset) {
				// if it does not exist, it return nil as error
				// so we need to add it to the fake client to be able to pass the existence check
				err := createConfigMap(clientset, "error-configmap", suite.namespace)
				require.NoError(suite.T(), err)

				clientset.PrependReactor("delete", "configmaps", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: k8s.ErrDeletingConfigmap.WithParams("error-configmap").
				Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			err := suite.client.DeleteConfigMap(context.Background(), tt.configMapName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func createConfigMap(clientset *fake.Clientset, name, namespace string) error {
	_, err := clientset.CoreV1().ConfigMaps(namespace).Create(context.Background(), &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}, metav1.CreateOptions{})
	return err
}
