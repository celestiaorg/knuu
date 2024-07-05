package k8s_test

import (
	"context"
	"errors"

	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

func (s *TestSuite) TestCreateNetworkPolicy() {
	tests := []struct {
		name               string
		npName             string
		selectorMap        map[string]string
		ingressSelectorMap map[string]string
		egressSelectorMap  map[string]string
		setupMock          func()
		expectedErr        error
	}{
		{
			name:        "successful creation",
			npName:      "test-np",
			selectorMap: map[string]string{"app": "test"},
			setupMock:   func() {},
			expectedErr: nil,
		},
		{
			name:        "client error",
			npName:      "error-np",
			selectorMap: map[string]string{"app": "error"},
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("create", "networkpolicies",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, k8s.ErrCreatingNetworkPolicy.WithParams("error-np").
								Wrap(errInternalServerError)
						})
			},
			expectedErr: k8s.ErrCreatingNetworkPolicy.WithParams("error-np").Wrap(errInternalServerError),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			err := s.client.CreateNetworkPolicy(context.Background(), tt.npName, tt.selectorMap, tt.ingressSelectorMap, tt.egressSelectorMap)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) TestDeleteNetworkPolicy() {
	tests := []struct {
		name        string
		npName      string
		setupMock   func()
		expectedErr error
	}{
		{
			name:   "successful deletion",
			npName: "existing-np",
			setupMock: func() {
				err := s.createNetworkPolicy("existing-np")
				s.Require().NoError(err)
			},
			expectedErr: nil,
		},
		{
			name:      "network policy not found",
			npName:    "non-existent-np",
			setupMock: func() {},
			expectedErr: k8s.ErrDeletingNetworkPolicy.WithParams("non-existent-np").
				Wrap(errors.New("networkpolicies \"non-existent-np\" not found")),
		},
		{
			name:   "client error",
			npName: "error-np",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "networkpolicies",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errInternalServerError
						})
			},
			expectedErr: k8s.ErrDeletingNetworkPolicy.WithParams("error-np").
				Wrap(errInternalServerError),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			err := s.client.DeleteNetworkPolicy(context.Background(), tt.npName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) TestGetNetworkPolicy() {
	tests := []struct {
		name        string
		npName      string
		setupMock   func()
		expectedErr error
		expectedNP  *v1.NetworkPolicy
	}{
		{
			name:   "successful retrieval",
			npName: "existing-np",
			setupMock: func() {
				err := s.createNetworkPolicy("existing-np")
				s.Require().NoError(err)
			},
			expectedErr: nil,
			expectedNP: &v1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-np",
					Namespace: s.namespace,
				},
			},
		},
		{
			name:   "network policy not found",
			npName: "non-existent-np",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "networkpolicies",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
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
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "networkpolicies",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errInternalServerError
						})
			},
			expectedErr: k8s.ErrGettingNetworkPolicy.WithParams("error-np").
				Wrap(errInternalServerError),
			expectedNP: nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			np, err := s.client.GetNetworkPolicy(context.Background(), tt.npName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
			s.Assert().EqualValues(tt.expectedNP, np)
		})
	}
}

func (s *TestSuite) TestNetworkPolicyExists() {
	tests := []struct {
		name          string
		npName        string
		setupMock     func()
		expectedExist bool
	}{
		{
			name:   "network policy exists",
			npName: "existing-np",
			setupMock: func() {
				err := s.createNetworkPolicy("existing-np")
				s.Require().NoError(err)
			},
			expectedExist: true,
		},
		{
			name:          "network policy does not exist",
			npName:        "non-existent-np",
			setupMock:     func() {},
			expectedExist: false,
		},
		{
			name:   "client error",
			npName: "error-np",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "networkpolicies",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errInternalServerError
						})
			},
			expectedExist: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			exists := s.client.NetworkPolicyExists(context.Background(), tt.npName)
			s.Assert().Equal(tt.expectedExist, exists)
		})
	}
}
