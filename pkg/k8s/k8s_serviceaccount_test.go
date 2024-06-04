package k8s_test

import (
	"context"
	"errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func (suite *TestSuite) TestCreateServiceAccount() {
	tests := []struct {
		name        string
		saName      string
		labels      map[string]string
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name:        "successful creation",
			saName:      "test-sa",
			labels:      map[string]string{"app": "test"},
			setupMock:   func(clientset *fake.Clientset) {},
			expectedErr: nil,
		},
		{
			name:   "client error",
			saName: "error-sa",
			labels: map[string]string{"app": "error"},
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("create", "serviceaccounts", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: errors.New("internal server error"),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			err := suite.client.CreateServiceAccount(context.Background(), tt.saName, tt.labels)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.Equal(suite.T(), tt.expectedErr.Error(), err.Error())
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func (suite *TestSuite) TestDeleteServiceAccount() {
	tests := []struct {
		name        string
		saName      string
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name:   "successful deletion",
			saName: "test-sa",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("delete", "serviceaccounts", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, nil
				})
			},
			expectedErr: nil,
		},
		{
			name:   "client error",
			saName: "error-sa",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("delete", "serviceaccounts", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: errors.New("internal server error"),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			err := suite.client.DeleteServiceAccount(context.Background(), tt.saName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.Equal(suite.T(), tt.expectedErr.Error(), err.Error())
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}
