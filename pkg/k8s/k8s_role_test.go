package k8s_test

import (
	"context"
	"errors"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

func (s *TestSuite) TestCreateRole() {
	tests := []struct {
		name        string
		roleName    string
		labels      map[string]string
		policyRules []rbacv1.PolicyRule
		setupMock   func()
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
			setupMock:   func() {},
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
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("create", "roles",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: errors.New("internal server error"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			err := s.client.CreateRole(context.Background(), tt.roleName, tt.labels, tt.policyRules)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().Equal(tt.expectedErr.Error(), err.Error())
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) TestDeleteRole() {
	tests := []struct {
		name        string
		roleName    string
		setupMock   func()
		expectedErr error
	}{
		{
			name:     "successful deletion",
			roleName: "test-role",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "roles",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, nil
						})
			},
			expectedErr: nil,
		},
		{
			name:     "client error",
			roleName: "error-role",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "roles",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: errors.New("internal server error"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			err := s.client.DeleteRole(context.Background(), tt.roleName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().Equal(tt.expectedErr.Error(), err.Error())
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) TestCreateClusterRole() {
	tests := []struct {
		name        string
		roleName    string
		labels      map[string]string
		policyRules []rbacv1.PolicyRule
		setupMock   func()
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
			setupMock:   func() {},
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
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "clusterroles",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, nil
						})
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("create", "clusterroles",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrClusterRoleAlreadyExists.WithParams("error-cluster-role").Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			err := s.client.CreateClusterRole(context.Background(), tt.roleName, tt.labels, tt.policyRules)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().Equal(tt.expectedErr.Error(), err.Error())
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) TestDeleteClusterRole() {
	tests := []struct {
		name        string
		roleName    string
		setupMock   func()
		expectedErr error
	}{
		{
			name:     "successful deletion",
			roleName: "test-cluster-role",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "clusterroles",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, nil
						})
			},
			expectedErr: nil,
		},
		{
			name:     "client error",
			roleName: "error-cluster-role",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "clusterroles",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: errors.New("internal server error"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			err := s.client.DeleteClusterRole(context.Background(), tt.roleName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().Equal(tt.expectedErr.Error(), err.Error())
				return
			}

			s.Require().NoError(err)
		})
	}
}
