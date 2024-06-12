package k8s_test

import (
	"context"
	"errors"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

func (s *TestSuite) TestGetConfigMap() {
	tests := []struct {
		name          string
		configMapName string
		setupMock     func()
		expectedErr   error
		expectedCM    *v1.ConfigMap
	}{
		{
			name:          "successful retrieval",
			configMapName: "test-configmap",
			setupMock: func() {
				err := s.createConfigMap("test-configmap")
				s.Require().NoError(err)
			},
			expectedErr: nil,
			expectedCM: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-configmap",
					Namespace: s.namespace,
				},
			},
		},
		{
			name:          "configmap not found",
			configMapName: "non-existent-configmap",
			setupMock: func() {
				// No setup needed for this case
			},
			expectedErr: k8s.ErrGettingConfigmap.WithParams("non-existent-configmap").
				Wrap(errors.New("configmaps \"non-existent-configmap\" not found")),
			expectedCM: nil,
		},
		{
			name:          "client error",
			configMapName: "error-configmap",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "configmaps",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrGettingConfigmap.WithParams("error-configmap").
				Wrap(errors.New("internal server error")),
			expectedCM: nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			cm, err := s.client.GetConfigMap(context.Background(), tt.configMapName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
			s.Assert().EqualValues(tt.expectedCM, cm)
		})
	}
}

func (s *TestSuite) TestConfigMapExists() {
	tests := []struct {
		name          string
		configMapName string
		setupMock     func()
		expectedExist bool
		expectedErr   error
	}{
		{
			name:          "configmap exists",
			configMapName: "existing-configmap",
			setupMock: func() {
				err := s.createConfigMap("existing-configmap")
				s.Require().NoError(err)
			},
			expectedExist: true,
			expectedErr:   nil,
		},
		{
			name:          "configmap does not exist",
			configMapName: "non-existent-configmap",
			setupMock:     func() {},
			expectedExist: false,
			expectedErr:   nil,
		},
		{
			name:          "client error",
			configMapName: "error-configmap",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "configmaps",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedExist: false,
			expectedErr: k8s.ErrGettingConfigmap.WithParams("error-configmap").
				Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			exists, err := s.client.ConfigMapExists(context.Background(), tt.configMapName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
			s.Assert().Equal(tt.expectedExist, exists)
		})
	}
}

func (s *TestSuite) TestCreateConfigMap() {
	tests := []struct {
		name        string
		configMap   *v1.ConfigMap
		setupMock   func()
		expectedErr error
	}{
		{
			name: "successful creation",
			configMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-configmap",
					Namespace: s.namespace,
				},
				Data: map[string]string{"key": "value"},
			},
			setupMock:   func() {},
			expectedErr: nil,
		},
		{
			name: "configmap already exists",
			configMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-configmap",
					Namespace: s.namespace,
				},
			},
			setupMock: func() {
				err := s.createConfigMap("existing-configmap")
				s.Require().NoError(err)
			},
			expectedErr: k8s.ErrConfigmapAlreadyExists.WithParams("existing-configmap").
				Wrap(errors.New("configmap already exists")),
		},
		{
			name: "client error",
			configMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "error-configmap",
					Namespace: s.namespace,
				},
			},
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("create", "configmaps",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrCreatingConfigmap.WithParams("error-configmap").
				Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			cm, err := s.client.CreateConfigMap(context.Background(), tt.configMap.Name, tt.configMap.Labels, tt.configMap.Data)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
			s.Assert().EqualValues(tt.configMap, cm)
		})
	}
}

func (s *TestSuite) TestDeleteConfigMap() {
	tests := []struct {
		name          string
		configMapName string
		setupMock     func()
		expectedErr   error
	}{
		{
			name:          "successful deletion",
			configMapName: "existing-configmap",
			setupMock: func() {
				err := s.createConfigMap("existing-configmap")
				s.Require().NoError(err)
			},
			expectedErr: nil,
		},
		{
			name:          "configmap does not exist",
			configMapName: "non-existent-configmap",
			setupMock:     func() {},
			expectedErr: k8s.ErrConfigmapDoesNotExist.WithParams("non-existent-configmap").
				Wrap(errors.New("configmap does not exist")),
		},
		{
			name:          "client error",
			configMapName: "error-configmap",
			setupMock: func() {
				// if it does not exist, it return nil as error
				// so we need to add it to the fake client to be able to pass the existence check
				err := s.createConfigMap("error-configmap")
				s.Require().NoError(err)

				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "configmaps",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrDeletingConfigmap.WithParams("error-configmap").
				Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			err := s.client.DeleteConfigMap(context.Background(), tt.configMapName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
		})
	}
}
