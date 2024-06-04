package k8s_test

import (
	"context"
	"errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

func (suite *TestSuite) TestCreateNamespace() {
	tests := []struct {
		name        string
		namespace   string
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name:        "successful creation",
			namespace:   "new-namespace",
			setupMock:   func(clientset *fake.Clientset) {},
			expectedErr: nil,
		},
		{
			name:      "namespace already exists",
			namespace: "existing-namespace",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("create", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, nil
				})
			},
			expectedErr: nil,
		},
		{
			name:      "client error",
			namespace: "error-namespace",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("create", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, k8s.ErrCreatingNamespace.WithParams("error-namespace").
						Wrap(errors.New("internal server error"))
				})
			},
			expectedErr: k8s.ErrCreatingNamespace.WithParams("error-namespace").
				Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			err := suite.client.CreateNamespace(context.Background(), tt.namespace)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func (suite *TestSuite) TestDeleteNamespace() {
	tests := []struct {
		name        string
		namespace   string
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name:      "successful deletion",
			namespace: "existing-namespace",
			setupMock: func(clientset *fake.Clientset) {
				err := createNamespace(clientset, "existing-namespace")
				require.NoError(suite.T(), err)
			},
			expectedErr: nil,
		},
		{
			name:      "namespace not found",
			namespace: "non-existent-namespace",
			setupMock: func(clientset *fake.Clientset) {},
			expectedErr: k8s.ErrDeletingNamespace.WithParams("non-existent-namespace").
				Wrap(errors.New("namespaces \"non-existent-namespace\" not found")),
		},
		{
			name:      "client error",
			namespace: "error-namespace",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("delete", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: k8s.ErrDeletingNamespace.WithParams("error-namespace").
				Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			err := suite.client.DeleteNamespace(context.Background(), tt.namespace)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func (suite *TestSuite) TestGetNamespace() {
	tests := []struct {
		name        string
		namespace   string
		setupMock   func(*fake.Clientset)
		expectedErr error
		expectedNS  *corev1.Namespace
	}{
		{
			name:      "successful retrieval",
			namespace: "existing-namespace",
			setupMock: func(clientset *fake.Clientset) {
				err := createNamespace(clientset, "existing-namespace")
				require.NoError(suite.T(), err)
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
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
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
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: k8s.ErrGettingNamespace.WithParams("error-namespace").
				Wrap(errors.New("internal server error")),
			expectedNS: nil,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			ns, err := suite.client.GetNamespace(context.Background(), tt.namespace)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
			assert.EqualValues(suite.T(), tt.expectedNS, ns)
		})
	}
}

func (suite *TestSuite) TestNamespaceExists() {
	tests := []struct {
		name          string
		namespace     string
		setupMock     func(*fake.Clientset)
		expectedExist bool
	}{
		{
			name:      "namespace exists",
			namespace: "existing-namespace",
			setupMock: func(clientset *fake.Clientset) {
				err := createNamespace(clientset, "existing-namespace")
				require.NoError(suite.T(), err)
			},
			expectedExist: true,
		},
		{
			name:          "namespace does not exist",
			namespace:     "non-existent-namespace",
			setupMock:     func(clientset *fake.Clientset) {},
			expectedExist: false,
		},
		{
			name:      "client error",
			namespace: "error-namespace",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedExist: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			exists := suite.client.NamespaceExists(context.Background(), tt.namespace)
			assert.Equal(suite.T(), tt.expectedExist, exists)
		})
	}
}

func createNamespace(clientset *fake.Clientset, name string) error {
	_, err := clientset.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}, metav1.CreateOptions{})
	return err
}
