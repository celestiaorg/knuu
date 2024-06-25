package k8s_test

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	discfake "k8s.io/client-go/discovery/fake"
	dynfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

func (s *TestSuite) TestCreateCustomResource() {
	tests := []struct {
		name        string
		resource    *schema.GroupVersionResource
		obj         *unstructured.Unstructured
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
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"key": "value",
					},
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
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"key": "value",
					},
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
		s.Run(tt.name, func() {
			tt.setupMock(s.client.DynamicClient().(*dynfake.FakeDynamicClient))

			err := s.client.CreateCustomResource(context.Background(), "test-resource", tt.resource, tt.obj)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) TestCustomResourceDefinitionExists() {
	const (
		group        = "example.com"
		version      = "v1"
		resource     = "examples"
		groupVersion = "example.com/v1"
		kind         = "example-kind"
	)

	tests := []struct {
		name           string
		resource       *schema.GroupVersionResource
		setupMock      func(*discfake.FakeDiscovery)
		expectedExists bool
		expectedErr    error
	}{
		{
			name: "resource definition exists",
			resource: &schema.GroupVersionResource{
				Group:    group,
				Version:  version,
				Resource: resource,
			},
			setupMock: func(discoveryClient *discfake.FakeDiscovery) {
				discoveryClient.Resources = []*metav1.APIResourceList{
					{
						GroupVersion: groupVersion,
						APIResources: []metav1.APIResource{
							{
								Name: resource,
								// must be equal to the kind in the resource.Resource definition
								Kind: resource,
							},
						},
					},
				}
			},
			expectedExists: true,
			expectedErr:    nil,
		},
		{
			name: "resource definition does not exist",
			resource: &schema.GroupVersionResource{
				Group:    group,
				Version:  version,
				Resource: "nonexistent",
			},
			setupMock: func(discoveryClient *discfake.FakeDiscovery) {
				discoveryClient.Resources = []*metav1.APIResourceList{
					{
						GroupVersion: groupVersion,
						APIResources: []metav1.APIResource{},
					},
				}
			},
			expectedExists: false,
			expectedErr:    nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock(s.client.DiscoveryClient().(*discfake.FakeDiscovery))

			exists, err := s.client.CustomResourceDefinitionExists(context.Background(), tt.resource)
			fmt.Printf("err: %v\n", err)
			s.Assert().Equal(tt.expectedExists, exists)
			s.Assert().ErrorIs(err, tt.expectedErr)
		})
	}
}
