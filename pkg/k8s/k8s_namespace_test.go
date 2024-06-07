package k8s_test

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
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
								Wrap(errors.New("internal server error"))
						})
			},
			expectedErr: k8s.ErrCreatingNamespace.WithParams("error-namespace").
				Wrap(errors.New("internal server error")),
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
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrDeletingNamespace.WithParams("error-namespace").
				Wrap(errors.New("internal server error")),
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
		name        string
		namespace   string
		setupMock   func()
		expectedErr error
		expectedNS  *corev1.Namespace
	}{
		{
			name:      "successful retrieval",
			namespace: "existing-namespace",
			setupMock: func() {
				err := s.createNamespace("existing-namespace")
				s.Require().NoError(err)
			},
			expectedErr: nil,
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
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "namespaces",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("namespaces \"non-existent-namespace\" not found")
						})
			},
			expectedErr: k8s.ErrGettingNamespace.WithParams("non-existent-namespace").
				Wrap(errors.New("namespaces \"non-existent-namespace\" not found")),
			expectedNS: nil,
		},
		{
			name:      "client error",
			namespace: "error-namespace",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "namespaces",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrGettingNamespace.WithParams("error-namespace").
				Wrap(errors.New("internal server error")),
			expectedNS: nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			ns, err := s.client.GetNamespace(context.Background(), tt.namespace)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
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
	}{
		{
			name:      "namespace exists",
			namespace: "existing-namespace",
			setupMock: func() {
				err := s.createNamespace("existing-namespace")
				s.Require().NoError(err)
			},
			expectedExist: true,
		},
		{
			name:          "namespace does not exist",
			namespace:     "non-existent-namespace",
			setupMock:     func() {},
			expectedExist: false,
		},
		{
			name:      "client error",
			namespace: "error-namespace",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "namespaces",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedExist: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			exists := s.client.NamespaceExists(context.Background(), tt.namespace)
			s.Assert().Equal(tt.expectedExist, exists)
		})
	}
}

func (s *TestSuite) createNamespace(name string) error {
	_, err := s.client.Clientset().CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}, metav1.CreateOptions{})
	return err
}
