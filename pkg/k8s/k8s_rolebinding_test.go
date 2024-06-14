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

func (s *TestSuite) TestCreateRoleBinding() {
	tests := []struct {
		name            string
		roleBindingName string
		labels          map[string]string
		role            string
		serviceAccount  string
		setupMock       func()
		expectedErr     error
	}{
		{
			name:            "successful creation",
			roleBindingName: "test-rolebinding",
			labels:          map[string]string{"app": "test"},
			role:            "test-role",
			serviceAccount:  "test-sa",
			setupMock:       func() {},
			expectedErr:     nil,
		},
		{
			name:            "client error",
			roleBindingName: "error-rolebinding",
			labels:          map[string]string{"app": "error"},
			role:            "error-role",
			serviceAccount:  "error-sa",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("create", "rolebindings",
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

			err := s.client.CreateRoleBinding(context.Background(), tt.roleBindingName, tt.labels, tt.role, tt.serviceAccount)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().Equal(tt.expectedErr.Error(), err.Error())
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) TestDeleteRoleBinding() {
	tests := []struct {
		name        string
		bindingName string
		setupMock   func()
		expectedErr error
	}{
		{
			name:        "successful deletion",
			bindingName: "test-rolebinding",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "rolebindings",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, nil
						})
			},
			expectedErr: nil,
		},
		{
			name:        "client error",
			bindingName: "error-rolebinding",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "rolebindings",
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

			err := s.client.DeleteRoleBinding(context.Background(), tt.bindingName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().Equal(tt.expectedErr.Error(), err.Error())
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) TestCreateClusterRoleBinding() {
	tests := []struct {
		name           string
		bindingName    string
		labels         map[string]string
		clusterRole    string
		serviceAccount string
		setupMock      func()
		expectedErr    error
	}{
		{
			name:           "successful creation",
			bindingName:    "test-clusterrolebinding",
			labels:         map[string]string{"app": "test"},
			clusterRole:    "test-clusterrole",
			serviceAccount: "test-sa",
			setupMock:      func() {},
			expectedErr:    nil,
		},
		{
			name:           "cluster role binding already exists",
			bindingName:    "existing-clusterrolebinding",
			labels:         map[string]string{"app": "existing"},
			clusterRole:    "existing-clusterrole",
			serviceAccount: "existing-sa",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "clusterrolebindings",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
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
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "clusterrolebindings",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrClusterRoleBindingAlreadyExists.WithParams("error-clusterrolebinding").Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			err := s.client.CreateClusterRoleBinding(context.Background(), tt.bindingName, tt.labels, tt.clusterRole, tt.serviceAccount)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().Equal(tt.expectedErr.Error(), err.Error())
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) TestDeleteClusterRoleBinding() {
	tests := []struct {
		name        string
		bindingName string
		setupMock   func()
		expectedErr error
	}{
		{
			name:        "successful deletion",
			bindingName: "test-clusterrolebinding",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "clusterrolebindings",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, nil
						})
			},
			expectedErr: nil,
		},
		{
			name:        "client error",
			bindingName: "error-clusterrolebinding",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "clusterrolebindings",
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

			err := s.client.DeleteClusterRoleBinding(context.Background(), tt.bindingName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().Equal(tt.expectedErr.Error(), err.Error())
				return
			}

			s.Require().NoError(err)
		})
	}
}
