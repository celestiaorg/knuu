package k8s_test

import (
	"context"
	"errors"

	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

func (s *TestSuite) TestDaemonSetExists() {
	tests := []struct {
		name           string
		daemonSetName  string
		setupMock      func()
		expectedExists bool
		expectedErr    error
	}{
		{
			name:          "daemonset exists",
			daemonSetName: "existing-daemonset",
			setupMock: func() {
				s.Require().NoError(s.createDaemonSet("existing-daemonset"))
			},
			expectedExists: true,
			expectedErr:    nil,
		},
		{
			name:           "daemonset does not exist",
			daemonSetName:  "non-existent-daemonset",
			setupMock:      func() {},
			expectedExists: false,
			expectedErr:    nil,
		},
		{
			name:          "client error",
			daemonSetName: "error-daemonset",
			setupMock: func() {
				s.Require().NoError(s.createDaemonSet("error-daemonset"))
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "daemonsets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedExists: false,
			expectedErr:    k8s.ErrGettingDaemonset.WithParams("error-daemonset"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			exists, err := s.client.DaemonSetExists(context.Background(), tt.daemonSetName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
			s.Assert().Equal(tt.expectedExists, exists)
		})
	}
}

func (s *TestSuite) TestGetDaemonSet() {
	tests := []struct {
		name          string
		daemonSetName string
		setupMock     func()
		expectedErr   error
		expectedDS    *appv1.DaemonSet
	}{
		{
			name:          "successful retrieval",
			daemonSetName: "test-daemonset",
			setupMock: func() {
				s.Require().NoError(s.createDaemonSet("test-daemonset"))
			},
			expectedErr: nil,
			expectedDS: &appv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-daemonset",
					Namespace: s.namespace,
				},
			},
		},
		{
			name:          "daemonset not found",
			daemonSetName: "non-existent-daemonset",
			setupMock:     func() {},
			expectedErr:   k8s.ErrGettingDaemonset.Wrap(errors.New("daemonsets \"non-existent-daemonset\" not found")),
			expectedDS:    nil,
		},
		{
			name:          "client error",
			daemonSetName: "error-daemonset",
			setupMock: func() {
				s.Require().NoError(s.createDaemonSet("error-daemonset"))
				s.client.Clientset().(*fake.Clientset).PrependReactor("get", "daemonsets",
					func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
						return true, nil, errors.New("internal server error")
					})
			},
			expectedErr: k8s.ErrGettingDaemonset.WithParams("error-daemonset").Wrap(errors.New("internal server error")),
			expectedDS:  nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			ds, err := s.client.GetDaemonSet(context.Background(), tt.daemonSetName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
			s.Assert().EqualValues(tt.expectedDS, ds)
		})
	}
}

func (s *TestSuite) TestCreateDaemonSet() {
	tests := []struct {
		name           string
		daemonSetName  string
		labels         map[string]string
		initContainers []v1.Container
		containers     []v1.Container
		setupMock      func()
		expectedErr    error
		expectedDS     *appv1.DaemonSet
	}{
		{
			name:           "successful creation",
			daemonSetName:  "new-daemonset",
			labels:         map[string]string{"app": "test"},
			initContainers: []v1.Container{},
			containers: []v1.Container{
				{
					Name:  "container",
					Image: "nginx",
				},
			},
			setupMock:   func() {},
			expectedErr: nil,
			expectedDS: &appv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-daemonset",
					Namespace: s.namespace,
					Labels:    map[string]string{"app": "test"},
				},
				Spec: appv1.DaemonSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "test"},
						},
						Spec: v1.PodSpec{
							InitContainers: []v1.Container{},
							Containers: []v1.Container{
								{
									Name:  "container",
									Image: "nginx",
								},
							},
						},
					},
				},
			},
		},
		{
			name:           "client error",
			daemonSetName:  "error-daemonset",
			labels:         map[string]string{"app": "test"},
			initContainers: []v1.Container{},
			containers: []v1.Container{
				{
					Name:  "container",
					Image: "nginx",
				},
			},
			setupMock: func() {
				s.Require().NoError(s.createDaemonSet("error-daemonset"))
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("create", "daemonsets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrCreatingDaemonset.WithParams("error-daemonset").Wrap(errors.New("internal server error")),
			expectedDS:  nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			ds, err := s.client.CreateDaemonSet(context.Background(), tt.daemonSetName, tt.labels, tt.initContainers, tt.containers)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
			s.Assert().EqualValues(tt.expectedDS, ds)
		})
	}
}

func (s *TestSuite) TestUpdateDaemonSet() {
	tests := []struct {
		name           string
		daemonSetName  string
		labels         map[string]string
		initContainers []v1.Container
		containers     []v1.Container
		setupMock      func()
		expectedErr    error
		expectedDS     *appv1.DaemonSet
	}{
		{
			name:           "successful update",
			daemonSetName:  "existing-daemonset",
			labels:         map[string]string{"app": "test"},
			initContainers: []v1.Container{},
			containers: []v1.Container{
				{
					Name:  "container",
					Image: "nginx",
				},
			},
			setupMock: func() {
				_, err := s.client.Clientset().AppsV1().
					DaemonSets(s.namespace).
					Create(context.Background(), &appv1.DaemonSet{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "existing-daemonset",
							Namespace: s.namespace,
							Labels:    map[string]string{"app": "test"},
						},
						Spec: appv1.DaemonSetSpec{
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"app": "test"},
							},
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"app": "test"},
								},
								Spec: v1.PodSpec{
									InitContainers: []v1.Container{},
									Containers: []v1.Container{
										{
											Name:  "container",
											Image: "nginx",
										},
									},
								},
							},
						},
					}, metav1.CreateOptions{})
				s.Require().NoError(err)
			},
			expectedErr: nil,
			expectedDS: &appv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-daemonset",
					Namespace: s.namespace,
					Labels:    map[string]string{"app": "test"},
				},
				Spec: appv1.DaemonSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "test"},
						},
						Spec: v1.PodSpec{
							InitContainers: []v1.Container{},
							Containers: []v1.Container{
								{
									Name:  "container",
									Image: "nginx",
								},
							},
						},
					},
				},
			},
		},
		{
			name:           "client error",
			daemonSetName:  "error-daemonset",
			labels:         map[string]string{"app": "test"},
			initContainers: []v1.Container{},
			containers: []v1.Container{
				{
					Name:  "container",
					Image: "nginx",
				},
			},
			setupMock: func() {
				s.Require().NoError(s.createDaemonSet("error-daemonset"))
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("update", "daemonsets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrUpdatingDaemonset.WithParams("error-daemonset").Wrap(errors.New("internal server error")),
			expectedDS:  nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			ds, err := s.client.UpdateDaemonSet(context.Background(), tt.daemonSetName, tt.labels, tt.initContainers, tt.containers)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
			s.Assert().EqualValues(tt.expectedDS, ds)
		})
	}
}

func (s *TestSuite) TestDeleteDaemonSet() {
	tests := []struct {
		name          string
		daemonSetName string
		setupMock     func()
		expectedErr   error
	}{
		{
			name:          "successful deletion",
			daemonSetName: "existing-daemonset",
			setupMock: func() {
				s.Require().NoError(s.createDaemonSet("existing-daemonset"))
			},
			expectedErr: nil,
		},
		{
			name:          "daemonset does not exist",
			daemonSetName: "non-existent-daemonset",
			setupMock:     func() {},
			expectedErr:   k8s.ErrDeletingDaemonset.WithParams("non-existent-daemonset").Wrap(errors.New("daemonset does not exist")),
		},
		{
			name:          "client error",
			daemonSetName: "error-daemonset",
			setupMock: func() {
				s.Require().NoError(s.createDaemonSet("error-daemonset"))
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "daemonsets",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrDeletingDaemonset.WithParams("error-daemonset").Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			err := s.client.DeleteDaemonSet(context.Background(), tt.daemonSetName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
		})
	}
}
