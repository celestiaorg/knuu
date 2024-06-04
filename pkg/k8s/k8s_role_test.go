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

func (suite *TestSuite) TestCreateRole() {
	tests := []struct {
		name        string
		roleName    string
		labels      map[string]string
		policyRules []rbacv1.PolicyRule
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name:     "successful creation",
			roleName: "test-role",
			labels:   map[string]string{"app": "test"},
			policyRules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Verbs:     []string{"get", "list"},
					Resources: []string{"pods"},
				},
			},
			setupMock:   func(clientset *fake.Clientset) {},
			expectedErr: nil,
		},
		{
			name:     "client error",
			roleName: "error-role",
			labels:   map[string]string{"app": "error"},
			policyRules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Verbs:     []string{"get", "list"},
					Resources: []string{"pods"},
				},
			},
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("create", "roles", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: errors.New("internal server error"),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			err := suite.client.CreateRole(context.Background(), tt.roleName, tt.labels, tt.policyRules)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.Equal(suite.T(), tt.expectedErr.Error(), err.Error())
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func (suite *TestSuite) TestDeleteRole() {
	tests := []struct {
		name        string
		roleName    string
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name:     "successful deletion",
			roleName: "test-role",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("delete", "roles", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, nil
				})
			},
			expectedErr: nil,
		},
		{
			name:     "client error",
			roleName: "error-role",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("delete", "roles", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: errors.New("internal server error"),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			err := suite.client.DeleteRole(context.Background(), tt.roleName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.Equal(suite.T(), tt.expectedErr.Error(), err.Error())
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func (suite *TestSuite) TestCreateClusterRole() {
	tests := []struct {
		name        string
		roleName    string
		labels      map[string]string
		policyRules []rbacv1.PolicyRule
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name:     "successful creation",
			roleName: "test-cluster-role",
			labels:   map[string]string{"app": "test"},
			policyRules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Verbs:     []string{"get", "list"},
					Resources: []string{"pods"},
				},
			},
			setupMock:   func(clientset *fake.Clientset) {},
			expectedErr: nil,
		},
		{
			name:     "client error",
			roleName: "error-cluster-role",
			labels:   map[string]string{"app": "error"},
			policyRules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Verbs:     []string{"get", "list"},
					Resources: []string{"pods"},
				},
			},
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "clusterroles", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, nil
				})
				clientset.PrependReactor("create", "clusterroles", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: k8s.ErrClusterRoleAlreadyExists.WithParams("error-cluster-role").Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			err := suite.client.CreateClusterRole(context.Background(), tt.roleName, tt.labels, tt.policyRules)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.Equal(suite.T(), tt.expectedErr.Error(), err.Error())
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func (suite *TestSuite) TestDeleteClusterRole() {
	tests := []struct {
		name        string
		roleName    string
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name:     "successful deletion",
			roleName: "test-cluster-role",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("delete", "clusterroles", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, nil
				})
			},
			expectedErr: nil,
		},
		{
			name:     "client error",
			roleName: "error-cluster-role",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("delete", "clusterroles", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: errors.New("internal server error"),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			err := suite.client.DeleteClusterRole(context.Background(), tt.roleName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.Equal(suite.T(), tt.expectedErr.Error(), err.Error())
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}
