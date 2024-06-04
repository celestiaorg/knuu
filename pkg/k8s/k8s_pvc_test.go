package k8s_test

import (
	"context"
	"errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

func (suite *TestSuite) TestCreatePersistentVolumeClaim() {
	tests := []struct {
		name        string
		pvcName     string
		labels      map[string]string
		size        resource.Quantity
		setupMock   func()
		expectedErr error
	}{
		{
			name:        "successful creation",
			pvcName:     "test-pvc",
			labels:      map[string]string{"app": "test"},
			size:        resource.MustParse("1Gi"),
			setupMock:   func() {},
			expectedErr: nil,
		},
		{
			name:    "client error",
			pvcName: "error-pvc",
			labels:  map[string]string{"app": "error"},
			size:    resource.MustParse("1Gi"),
			setupMock: func() {
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("create", "persistentvolumeclaims",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrCreatingPersistentVolumeClaim.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock()

			err := suite.client.CreatePersistentVolumeClaim(context.Background(), tt.pvcName, tt.labels, tt.size)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func (suite *TestSuite) TestDeletePersistentVolumeClaim() {
	tests := []struct {
		name        string
		pvcName     string
		setupMock   func()
		expectedErr error
	}{
		{
			name:    "successful deletion",
			pvcName: "test-pvc",
			setupMock: func() {
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "persistentvolumeclaims",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &v1.PersistentVolumeClaim{
								ObjectMeta: metav1.ObjectMeta{
									Namespace: "test",
									Name:      "test-pvc",
								},
							}, nil
						})
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "persistentvolumeclaims",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, nil
						})
			},
			expectedErr: nil,
		},
		{
			name:    "pvc not found",
			pvcName: "missing-pvc",
			setupMock: func() {
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "persistentvolumeclaims",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("not found")
						})
			},
			expectedErr: nil, // it should skip deletion if pvc not found
		},
		{
			name:    "client error on delete",
			pvcName: "error-pvc",
			setupMock: func() {
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "persistentvolumeclaims",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &v1.PersistentVolumeClaim{
								ObjectMeta: metav1.ObjectMeta{
									Namespace: "test",
									Name:      "error-pvc",
								},
							}, nil
						})
				suite.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "persistentvolumeclaims",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrDeletingPersistentVolumeClaim.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock()

			err := suite.client.DeletePersistentVolumeClaim(context.Background(), tt.pvcName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}
