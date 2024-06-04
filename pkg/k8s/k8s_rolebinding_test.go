package k8s_test

import (
	"context"
	"errors"

	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func (suite *TestSuite) TestCreateRoleBinding() {
	tests := []struct {
		name            string
		roleBindingName string
		labels          map[string]string
		role            string
		serviceAccount  string
		setupMock       func(*fake.Clientset)
		expectedErr     error
	}{
		{
			name:            "successful creation",
			roleBindingName: "test-rolebinding",
			labels:          map[string]string{"app": "test"},
			role:            "test-role",
			serviceAccount:  "test-sa",
			setupMock:       func(clientset *fake.Clientset) {},
			expectedErr:     nil,
		},
		{
			name:            "client error",
			roleBindingName: "error-rolebinding",
			labels:          map[string]string{"app": "error"},
			role:            "error-role",
			serviceAccount:  "error-sa",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("create", "rolebindings", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: errors.New("internal server error"),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			err := suite.client.CreateRoleBinding(context.Background(), tt.roleBindingName, tt.labels, tt.role, tt.serviceAccount)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.Equal(suite.T(), tt.expectedErr.Error(), err.Error())
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func (suite *TestSuite) TestDeleteRoleBinding() {
	tests := []struct {
		name        string
		bindingName string
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name:        "successful deletion",
			bindingName: "test-rolebinding",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("delete", "rolebindings", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, nil
				})
			},
			expectedErr: nil,
		},
		{
			name:        "client error",
			bindingName: "error-rolebinding",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("delete", "rolebindings", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: errors.New("internal server error"),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			err := suite.client.DeleteRoleBinding(context.Background(), tt.bindingName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.Equal(suite.T(), tt.expectedErr.Error(), err.Error())
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func (suite *TestSuite) TestCreateClusterRoleBinding() {
	tests := []struct {
		name           string
		bindingName    string
		labels         map[string]string
		clusterRole    string
		serviceAccount string
		setupMock      func(*fake.Clientset)
		expectedErr    error
	}{
		{
			name:           "successful creation",
			bindingName:    "test-clusterrolebinding",
			labels:         map[string]string{"app": "test"},
			clusterRole:    "test-clusterrole",
			serviceAccount: "test-sa",
			setupMock:      func(clientset *fake.Clientset) {},
			expectedErr:    nil,
		},
		{
			name:           "cluster role binding already exists",
			bindingName:    "existing-clusterrolebinding",
			labels:         map[string]string{"app": "existing"},
			clusterRole:    "existing-clusterrole",
			serviceAccount: "existing-sa",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "clusterrolebindings", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, &rbacv1.ClusterRoleBinding{}, nil
				})
			},
			expectedErr: k8s.ErrClusterRoleBindingAlreadyExists.WithParams("existing-clusterrolebinding"),
		},
		{
			name:           "client error",
			bindingName:    "error-clusterrolebinding",
			labels:         map[string]string{"app": "error"},
			clusterRole:    "error-clusterrole",
			serviceAccount: "error-sa",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "clusterrolebindings", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: k8s.ErrClusterRoleBindingAlreadyExists.WithParams("error-clusterrolebinding").Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			err := suite.client.CreateClusterRoleBinding(context.Background(), tt.bindingName, tt.labels, tt.clusterRole, tt.serviceAccount)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.Equal(suite.T(), tt.expectedErr.Error(), err.Error())
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func (suite *TestSuite) TestDeleteClusterRoleBinding() {
	tests := []struct {
		name        string
		bindingName string
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name:        "successful deletion",
			bindingName: "test-clusterrolebinding",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("delete", "clusterrolebindings", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, nil
				})
			},
			expectedErr: nil,
		},
		{
			name:        "client error",
			bindingName: "error-clusterrolebinding",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("delete", "clusterrolebindings", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: errors.New("internal server error"),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			err := suite.client.DeleteClusterRoleBinding(context.Background(), tt.bindingName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.Equal(suite.T(), tt.expectedErr.Error(), err.Error())
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}
