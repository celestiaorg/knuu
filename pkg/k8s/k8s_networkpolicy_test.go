package k8s_test

import (
	"context"
	"errors"

	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func (suite *TestSuite) TestCreateNetworkPolicy() {
	tests := []struct {
		name               string
		npName             string
		selectorMap        map[string]string
		ingressSelectorMap map[string]string
		egressSelectorMap  map[string]string
		setupMock          func(*fake.Clientset)
		expectedErr        error
	}{
		{
			name:        "successful creation",
			npName:      "test-np",
			selectorMap: map[string]string{"app": "test"},
			setupMock:   func(clientset *fake.Clientset) {},
			expectedErr: nil,
		},
		{
			name:        "client error",
			npName:      "error-np",
			selectorMap: map[string]string{"app": "error"},
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("create", "networkpolicies", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, k8s.ErrCreatingNetworkPolicy.WithParams("error-np").
						Wrap(errors.New("internal server error"))
				})
			},
			expectedErr: k8s.ErrCreatingNetworkPolicy.WithParams("error-np").Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			err := suite.client.CreateNetworkPolicy(context.Background(), tt.npName, tt.selectorMap, tt.ingressSelectorMap, tt.egressSelectorMap)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func (suite *TestSuite) TestDeleteNetworkPolicy() {
	tests := []struct {
		name        string
		npName      string
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name:   "successful deletion",
			npName: "existing-np",
			setupMock: func(clientset *fake.Clientset) {
				err := createNetworkPolicy(clientset, "existing-np", suite.namespace)
				require.NoError(suite.T(), err)
			},
			expectedErr: nil,
		},
		{
			name:      "network policy not found",
			npName:    "non-existent-np",
			setupMock: func(clientset *fake.Clientset) {},
			expectedErr: k8s.ErrDeletingNetworkPolicy.WithParams("non-existent-np").
				Wrap(errors.New("networkpolicies \"non-existent-np\" not found")),
		},
		{
			name:   "client error",
			npName: "error-np",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("delete", "networkpolicies", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: k8s.ErrDeletingNetworkPolicy.WithParams("error-np").
				Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			err := suite.client.DeleteNetworkPolicy(context.Background(), tt.npName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func (suite *TestSuite) TestGetNetworkPolicy() {
	tests := []struct {
		name        string
		npName      string
		setupMock   func(*fake.Clientset)
		expectedErr error
		expectedNP  *v1.NetworkPolicy
	}{
		{
			name:   "successful retrieval",
			npName: "existing-np",
			setupMock: func(clientset *fake.Clientset) {
				err := createNetworkPolicy(clientset, "existing-np", suite.namespace)
				require.NoError(suite.T(), err)
			},
			expectedErr: nil,
			expectedNP: &v1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-np",
					Namespace: suite.namespace,
				},
			},
		},
		{
			name:   "network policy not found",
			npName: "non-existent-np",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "networkpolicies", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("networkpolicies \"non-existent-np\" not found")
				})
			},
			expectedErr: k8s.ErrGettingNetworkPolicy.WithParams("non-existent-np").
				Wrap(errors.New("networkpolicies \"non-existent-np\" not found")),
			expectedNP: nil,
		},
		{
			name:   "client error",
			npName: "error-np",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "networkpolicies", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: k8s.ErrGettingNetworkPolicy.WithParams("error-np").
				Wrap(errors.New("internal server error")),
			expectedNP: nil,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			np, err := suite.client.GetNetworkPolicy(context.Background(), tt.npName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
			assert.EqualValues(suite.T(), tt.expectedNP, np)
		})
	}
}

func (suite *TestSuite) TestNetworkPolicyExists() {
	tests := []struct {
		name          string
		npName        string
		setupMock     func(*fake.Clientset)
		expectedExist bool
	}{
		{
			name:   "network policy exists",
			npName: "existing-np",
			setupMock: func(clientset *fake.Clientset) {
				err := createNetworkPolicy(clientset, "existing-np", suite.namespace)
				require.NoError(suite.T(), err)
			},
			expectedExist: true,
		},
		{
			name:          "network policy does not exist",
			npName:        "non-existent-np",
			setupMock:     func(clientset *fake.Clientset) {},
			expectedExist: false,
		},
		{
			name:   "client error",
			npName: "error-np",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "networkpolicies", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedExist: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			exists := suite.client.NetworkPolicyExists(context.Background(), tt.npName)
			assert.Equal(suite.T(), tt.expectedExist, exists)
		})
	}
}

func createNetworkPolicy(clientset *fake.Clientset, name, namespace string) error {
	_, err := clientset.NetworkingV1().NetworkPolicies(namespace).Create(context.Background(), &v1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}, metav1.CreateOptions{})
	return err
}
