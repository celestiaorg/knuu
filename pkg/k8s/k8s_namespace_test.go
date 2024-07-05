package k8s_test

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

func (s *TestSuite) TestCreateNamespace() {
	tests := []struct {
		name        string
		namespace   string
		setupMock   func()
		expectedErr error
	}{
		{
			name:        "successful creation",
			namespace:   "new-namespace",
			setupMock:   func() {},
			expectedErr: nil,
		},
		{
			name:      "namespace already exists",
			namespace: "existing-namespace",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("create", "namespaces",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, nil
						})
			},
			expectedErr: nil,
		},
		{
			name:      "client error",
			namespace: "error-namespace",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("create", "namespaces",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, k8s.ErrCreatingNamespace.WithParams("error-namespace").
								Wrap(errInternalServerError)
						})
			},
			expectedErr: k8s.ErrCreatingNamespace.WithParams("error-namespace").
				Wrap(errInternalServerError),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			err := s.client.CreateNamespace(context.Background(), tt.namespace)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) TestDeleteNamespace() {
	tests := []struct {
		name        string
		namespace   string
		setupMock   func()
		expectedErr error
	}{
		{
			name:      "successful deletion",
			namespace: "existing-namespace",
			setupMock: func() {
				err := s.createNamespace("existing-namespace")
				s.Require().NoError(err)
			},
			expectedErr: nil,
		},
		{
			name:      "namespace not found",
			namespace: "non-existent-namespace",
			setupMock: func() {},
			expectedErr: k8s.ErrDeletingNamespace.WithParams("non-existent-namespace").
				Wrap(errors.New("namespaces \"non-existent-namespace\" not found")),
		},
		{
			name:      "client error",
			namespace: "error-namespace",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "namespaces",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errInternalServerError
						})
			},
			expectedErr: k8s.ErrDeletingNamespace.WithParams("error-namespace").
				Wrap(errInternalServerError),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			err := s.client.DeleteNamespace(context.Background(), tt.namespace)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) TestGetNamespace() {
	tests := []struct {
		name       string
		namespace  string
		setupMock  func()
		assertErr  func(err error)
		expectedNS *corev1.Namespace
	}{
		{
			name:      "successful retrieval",
			namespace: "existing-namespace",
			setupMock: func() {
				err := s.createNamespace("existing-namespace")
				s.Require().NoError(err)
			},
			assertErr: func(err error) {
				s.Require().NoError(err)
			},
			expectedNS: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing-namespace",
				},
			},
		},
		{
			name:      "namespace not found",
			namespace: "non-existent-namespace",
			setupMock: func() {
				// no need to mock
			},
			assertErr: func(err error) {
				s.Require().Error(err)
				s.Assert().True(apierrs.IsNotFound(err))
			},
			expectedNS: nil,
		},
		{
			name:      "client error",
			namespace: "error-namespace",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "namespaces",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errInternalServerError
						})
			},
			assertErr: func(err error) {
				s.Require().Error(err)
				s.Assert().Equal(err.Error(), "internal server error")
			},
			expectedNS: nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			ns, err := s.client.GetNamespace(context.Background(), tt.namespace)
			tt.assertErr(err)
			s.Assert().EqualValues(tt.expectedNS, ns)
		})
	}
}

func (s *TestSuite) TestNamespaceExists() {
	tests := []struct {
		name          string
		namespace     string
		setupMock     func()
		expectedExist bool
		expectedErr   error
	}{
		{
			name:      "namespace exists",
			namespace: "existing-namespace",
			setupMock: func() {
				err := s.createNamespace("existing-namespace")
				s.Require().NoError(err)
			},
			expectedExist: true,
			expectedErr:   nil,
		},
		{
			name:      "namespace does not exist",
			namespace: "non-existent-namespace",
			setupMock: func() {
				// no mock needed as the namespace does not exist
			},
			expectedExist: false,
			expectedErr:   nil,
		},
		{
			name:      "client error",
			namespace: "error-namespace",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "namespaces",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errInternalServerError
						})
			},
			expectedExist: false,
			expectedErr:   errInternalServerError,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			exists, err := s.client.NamespaceExists(context.Background(), tt.namespace)
			s.Assert().Equal(tt.expectedExist, exists)
			if tt.expectedErr == nil {
				s.Assert().NoError(err)
				return
			}

			s.Assert().Error(err)
			s.Assert().ErrorIs(err, tt.expectedErr)
		})
	}
}
