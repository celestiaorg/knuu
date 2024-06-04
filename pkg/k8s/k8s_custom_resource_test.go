package k8s_test

import (
	"context"
	"errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	discfake "k8s.io/client-go/discovery/fake"
	dynfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

func (suite *TestSuite) TestCreateCustomResource() {
	tests := []struct {
		name        string
		resource    *schema.GroupVersionResource
		obj         *map[string]interface{}
		setupMock   func(*dynfake.FakeDynamicClient)
		expectedErr error
	}{
		{
			name: "successful creation",
			resource: &schema.GroupVersionResource{
				Group:    "example.com",
				Version:  "v1",
				Resource: "examples",
			},
			obj: &map[string]interface{}{
				"spec": map[string]interface{}{
					"key": "value",
				},
			},
			setupMock:   func(dynamicClient *dynfake.FakeDynamicClient) {},
			expectedErr: nil,
		},
		{
			name: "client error",
			resource: &schema.GroupVersionResource{
				Group:    "example.com",
				Version:  "v1",
				Resource: "examples",
			},
			obj: &map[string]interface{}{
				"spec": map[string]interface{}{
					"key": "value",
				},
			},
			setupMock: func(dynamicClient *dynfake.FakeDynamicClient) {
				dynamicClient.PrependReactor("create", "examples",
					func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
						return true, nil, errors.New("internal server error")
					})
			},
			expectedErr: k8s.ErrCreatingCustomResource.WithParams("examples").
				Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.DynamicClient().(*dynfake.FakeDynamicClient))

			err := suite.client.CreateCustomResource(context.Background(), "test-resource", tt.resource, tt.obj)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func (suite *TestSuite) TestCustomResourceDefinitionExists() {
	tests := []struct {
		name           string
		resource       *schema.GroupVersionResource
		setupMock      func(*discfake.FakeDiscovery)
		expectedExists bool
	}{
		{
			name: "resource definition exists",
			resource: &schema.GroupVersionResource{
				Group:    "example.com",
				Version:  "v1",
				Resource: "example-kind",
			},
			setupMock: func(discoveryClient *discfake.FakeDiscovery) {
				discoveryClient.Resources = []*metav1.APIResourceList{
					{
						GroupVersion: "example.com/v1",
						APIResources: []metav1.APIResource{
							{
								Name: "examples",
								// must be equal to the kind in the resource.Resource definition
								Kind: "example-kind",
							},
						},
					},
				}
			},
			expectedExists: true,
		},
		{
			name: "resource definition does not exist",
			resource: &schema.GroupVersionResource{
				Group:    "example.com",
				Version:  "v1",
				Resource: "nonexistent",
			},
			setupMock: func(discoveryClient *discfake.FakeDiscovery) {
				discoveryClient.Resources = []*metav1.APIResourceList{
					{
						GroupVersion: "example.com/v1",
						APIResources: []metav1.APIResource{},
					},
				}
			},
			expectedExists: false,
		},
		{
			name: "discovery client error",
			resource: &schema.GroupVersionResource{
				Group:    "example.com",
				Version:  "v1",
				Resource: "examples",
			},
			setupMock: func(discoveryClient *discfake.FakeDiscovery) {
				discoveryClient.PrependReactor("get", "serverresourcesforgroupversion",
					func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
						return true, nil, errors.New("internal server error")
					})
			},
			expectedExists: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.DiscoveryClient().(*discfake.FakeDiscovery))

			exists := suite.client.CustomResourceDefinitionExists(context.Background(), tt.resource)
			assert.Equal(suite.T(), tt.expectedExists, exists)
		})
	}
}
