package k8s_test

import (
	"context"
	"errors"
	"net"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

func (s *TestSuite) TestGetService() {
	tests := []struct {
		name        string
		svcName     string
		setupMock   func()
		expectedErr error
	}{
		{
			name:    "successful retrieval",
			svcName: "test-service",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "services",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &v1.Service{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-service",
									Namespace: s.namespace,
								},
							}, nil
						})
			},
			expectedErr: nil,
		},
		{
			name:    "client error",
			svcName: "error-service",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "services",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrGettingService.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			svc, err := s.client.GetService(context.Background(), tt.svcName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().Equal(tt.expectedErr.Error(), err.Error())
				return
			}

			s.Require().NoError(err)
			s.Assert().Equal(tt.svcName, svc.Name)
		})
	}
}

func (s *TestSuite) TestCreateService() {
	tests := []struct {
		name        string
		svcName     string
		labels      map[string]string
		selectorMap map[string]string
		portsTCP    []int
		portsUDP    []int
		setupMock   func()
		expectedErr error
	}{
		{
			name:        "successful creation",
			svcName:     "test-service",
			labels:      map[string]string{"app": "test"},
			selectorMap: map[string]string{"app": "test"},
			portsTCP:    []int{80},
			portsUDP:    []int{53},
			setupMock:   func() {},
			expectedErr: nil,
		},
		{
			name:        "client error",
			svcName:     "error-service",
			labels:      map[string]string{"app": "error"},
			selectorMap: map[string]string{"app": "error"},
			portsTCP:    []int{80},
			portsUDP:    []int{53},
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("create", "services",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrCreatingService.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			svc, err := s.client.CreateService(context.Background(), tt.svcName, tt.labels, tt.selectorMap, tt.portsTCP, tt.portsUDP)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().Equal(tt.expectedErr.Error(), err.Error())
				return
			}

			s.Require().NoError(err)
			s.Assert().Equal(tt.svcName, svc.Name)
		})
	}
}

func (s *TestSuite) TestPatchService() {
	tests := []struct {
		name        string
		svcName     string
		labels      map[string]string
		selectorMap map[string]string
		portsTCP    []int
		portsUDP    []int
		setupMock   func()
		expectedErr error
	}{
		{
			name:        "successful patch",
			svcName:     "test-service",
			labels:      map[string]string{"app": "test"},
			selectorMap: map[string]string{"app": "test"},
			portsTCP:    []int{80},
			portsUDP:    []int{53},
			setupMock: func() {
				err := s.createService("test-service")
				s.Require().NoError(err)
			},
			expectedErr: nil,
		},
		{
			name:        "client error",
			svcName:     "error-service",
			labels:      map[string]string{"app": "error"},
			selectorMap: map[string]string{"app": "error"},
			portsTCP:    []int{80},
			portsUDP:    []int{53},
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("update", "services",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrPatchingService.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			svc, err := s.client.PatchService(context.Background(), tt.svcName, tt.labels, tt.selectorMap, tt.portsTCP, tt.portsUDP)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().Equal(tt.expectedErr.Error(), err.Error())
				return
			}

			s.Require().NoError(err)
			s.Assert().Equal(tt.svcName, svc.Name)
		})
	}
}

func (s *TestSuite) TestDeleteService() {
	tests := []struct {
		name        string
		svcName     string
		setupMock   func()
		expectedErr error
	}{
		{
			name:    "successful deletion",
			svcName: "test-service",
			setupMock: func() {
				err := s.createService("test-service")
				s.Require().NoError(err)
			},
			expectedErr: nil,
		},
		{
			name:    "client error",
			svcName: "error-service",
			setupMock: func() {
				err := s.createService("error-service")
				s.Require().NoError(err)

				s.client.Clientset().(*fake.Clientset).
					PrependReactor("delete", "services",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrDeletingService.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			err := s.client.DeleteService(context.Background(), tt.svcName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().Equal(tt.expectedErr.Error(), err.Error())
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) TestGetServiceIP() {
	tests := []struct {
		name        string
		svcName     string
		setupMock   func()
		expectedIP  string
		expectedErr error
	}{
		{
			name:    "successful retrieval",
			svcName: "test-service",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "services",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &v1.Service{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-service",
									Namespace: s.namespace,
								},
								Spec: v1.ServiceSpec{
									ClusterIP: "10.0.0.1",
								},
							}, nil
						})
			},
			expectedIP:  "10.0.0.1",
			expectedErr: nil,
		},
		{
			name:    "client error",
			svcName: "error-service",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "services",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedIP:  "",
			expectedErr: k8s.ErrGettingService.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			ip, err := s.client.GetServiceIP(context.Background(), tt.svcName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().Equal(tt.expectedErr.Error(), err.Error())
				return
			}

			s.Require().NoError(err)
			s.Assert().Equal(tt.expectedIP, ip)
		})
	}
}

func (s *TestSuite) TestWaitForService() {
	tests := []struct {
		name            string
		svcName         string
		setupMock       func()
		serviceEndpoint string
		expectedErr     error
	}{
		{
			name:            "successful wait load balancer",
			svcName:         "test-service",
			serviceEndpoint: "127.0.0.1:8171",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "services",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &v1.Service{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-service",
									Namespace: s.namespace,
								},
								Spec: v1.ServiceSpec{
									Type: v1.ServiceTypeLoadBalancer,
									Ports: []v1.ServicePort{
										{
											Port: 8171,
										},
									},
								},
								Status: v1.ServiceStatus{
									LoadBalancer: v1.LoadBalancerStatus{
										Ingress: []v1.LoadBalancerIngress{
											{
												IP: "127.0.0.1",
											},
										},
									},
								},
							}, nil
						})
			},
			expectedErr: nil,
		},
		{
			name:            "successful wait node port",
			svcName:         "test-service",
			serviceEndpoint: "127.0.0.1:8172",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "services",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &v1.Service{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-service",
									Namespace: s.namespace,
								},
								Spec: v1.ServiceSpec{
									Type: v1.ServiceTypeNodePort,
									Ports: []v1.ServicePort{
										{
											NodePort: 8172,
										},
									},
								},
							}, nil
						})
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("list", "nodes",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &v1.NodeList{
								Items: []v1.Node{
									{
										ObjectMeta: metav1.ObjectMeta{
											Name: "test-node",
										},
										Status: v1.NodeStatus{
											Addresses: []v1.NodeAddress{
												{
													Address: "127.0.0.1",
													Type:    v1.NodeExternalIP,
												},
											},
										},
									},
								},
							}, nil
						})
			},
			expectedErr: nil,
		},
		{
			name:            "successful wait cluster IP",
			svcName:         "test-service",
			serviceEndpoint: "127.0.0.1:8173",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "services",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &v1.Service{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-service",
									Namespace: s.namespace,
								},
								Spec: v1.ServiceSpec{
									ExternalIPs: []string{"127.0.0.1"},
									ClusterIP:   "127.0.0.1",
									Ports: []v1.ServicePort{
										{
											Port: 8173,
										},
									},
								},
							}, nil
						})
			},
			expectedErr: nil,
		},
		{
			name:    "context canceled",
			svcName: "canceled-service",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "services",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &v1.Service{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "canceled-service",
									Namespace: s.namespace,
								},
							}, nil
						})
			},
			expectedErr: k8s.ErrTimeoutWaitingForServiceReady,
		},
		{
			name:    "client error",
			svcName: "error-service",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "services",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedErr: k8s.ErrCheckingServiceReady.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.T().Parallel()
			tt.setupMock()

			if tt.serviceEndpoint != "" {
				listener, err := startDummyServer(tt.serviceEndpoint)
				s.Require().NoError(err)
				defer listener.Close()
			}

			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			err := s.client.WaitForService(ctx, tt.svcName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, tt.expectedErr)
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *TestSuite) TestGetServiceEndpoint() {
	tests := []struct {
		name        string
		svcName     string
		setupMock   func()
		expectedEP  string
		expectedErr error
	}{
		{
			name:    "successful retrieval for ClusterIP",
			svcName: "test-service",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "services",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, &v1.Service{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-service",
									Namespace: s.namespace,
								},
								Spec: v1.ServiceSpec{
									ClusterIP: "10.0.0.1",
									Ports: []v1.ServicePort{
										{
											Port: 80,
										},
									},
									Type: v1.ServiceTypeClusterIP,
								},
							}, nil
						})
			},
			expectedEP:  "10.0.0.1:80",
			expectedErr: nil,
		},
		{
			name:    "client error",
			svcName: "error-service",
			setupMock: func() {
				s.client.Clientset().(*fake.Clientset).
					PrependReactor("get", "services",
						func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
							return true, nil, errors.New("internal server error")
						})
			},
			expectedEP:  "",
			expectedErr: k8s.ErrGettingService.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			ep, err := s.client.GetServiceEndpoint(context.Background(), tt.svcName)
			if tt.expectedErr != nil {
				s.Require().Error(err)
				s.Assert().Equal(tt.expectedErr.Error(), err.Error())
				return
			}

			s.Require().NoError(err)
			s.Assert().Equal(tt.expectedEP, ep)
		})
	}
}

func (s *TestSuite) createService(name string) error {
	_, err := s.client.Clientset().CoreV1().Services(s.namespace).Create(context.Background(), &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.namespace,
		},
	}, metav1.CreateOptions{})
	return err
}

func startDummyServer(address string) (net.Listener, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	return listener, nil
}
