package k8s_test

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func (s *TestSuite) TestCreateServiceAccount() {
	tests := []struct {
		name        string
		saName      string
		labels      map[string]string
		setupMock   func()
		expectedErr error
	}{
		{
			name:        "successful creation",
			saName:      "test-sa",
			labels:      map[string]string{"app": "test"},
			setupMock:   func() {},
			expectedErr: nil,
		},
		{
			name:   "client error",
			saName: "error-sa",
			labels: map[string]string{"app": "error"},
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("create", "serviceaccounts",
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

			err := s.client.CreateServiceAccount(context.Background(), tt.saName, tt.labels)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().Equal(tt.expectedErr.Error(), err.Error())
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) TestDeleteServiceAccount() {
	tests := []struct {
		name        string
		saName      string
		setupMock   func()
		expectedErr error
	}{
		{
			name:   "successful deletion",
			saName: "test-sa",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "serviceaccounts",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, nil
						})
			},
			expectedErr: nil,
		},
		{
			name:   "client error",
			saName: "error-sa",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "serviceaccounts",
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

			err := s.client.DeleteServiceAccount(context.Background(), tt.saName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().Equal(tt.expectedErr.Error(), err.Error())
				return
			}

			s.Require().NoError(err)
		})
	}
}
