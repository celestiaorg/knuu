package k8s_test

import (
	"context"
	"errors"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

func (suite *TestSuite) TestWaitForDeployment() {
	tests := []struct {
		name           string
		deploymentName string
		setupMock      func()
		expectedErr    error
	}{
		{
			name:           "deployment becomes ready",
			deploymentName: "ready-deployment",
			setupMock: func() {
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "deployments",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &appsv1.Deployment{
								Status: appsv1.DeploymentStatus{
									ReadyReplicas: 1,
								},
							}, nil
						})
			},
			expectedErr: nil,
		},
		{
			name:           "deployment not found",
			deploymentName: "non-existent-deployment",
			setupMock: func() {
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "deployments",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("deployments \"non-existent-deployment\" not found")
						})
			},
			expectedErr: k8s.ErrWaitingForDeployment.WithParams("non-existent-deployment").
				Wrap(errors.New("deployments \"non-existent-deployment\" not found")),
		},
		{
			name:           "client error",
			deploymentName: "error-deployment",
			setupMock: func() {
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "deployments",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrWaitingForDeployment.WithParams("error-deployment").
				Wrap(errors.New("internal server error")),
		},
		{
			name:           "context timeout",
			deploymentName: "timeout-deployment",
			setupMock: func() {
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "deployments",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &appsv1.Deployment{
								Status: appsv1.DeploymentStatus{
									ReadyReplicas: 0,
								},
							}, nil
						})
			},
			expectedErr: k8s.ErrWaitingForDeployment.WithParams("timeout-deployment").Wrap(context.DeadlineExceeded),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.T().Parallel()
			tt.setupMock()

			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			err := suite.client.WaitForDeployment(ctx, tt.deploymentName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}
