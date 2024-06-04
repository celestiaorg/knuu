package k8s_test

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

func (suite *TestSuite) TestGetService() {
	tests := []struct {
		name        string
		svcName     string
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name:    "successful retrieval",
			svcName: "test-service",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, &v1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-service",
							Namespace: suite.namespace,
						},
					}, nil
				})
			},
			expectedErr: nil,
		},
		{
			name:    "client error",
			svcName: "error-service",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: k8s.ErrGettingService.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			svc, err := suite.client.GetService(context.Background(), tt.svcName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.Equal(suite.T(), tt.expectedErr.Error(), err.Error())
				return
			}

			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), tt.svcName, svc.Name)
		})
	}
}

func (suite *TestSuite) TestCreateService() {
	tests := []struct {
		name        string
		svcName     string
		labels      map[string]string
		selectorMap map[string]string
		portsTCP    []int
		portsUDP    []int
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name:        "successful creation",
			svcName:     "test-service",
			labels:      map[string]string{"app": "test"},
			selectorMap: map[string]string{"app": "test"},
			portsTCP:    []int{80},
			portsUDP:    []int{53},
			setupMock:   func(clientset *fake.Clientset) {},
			expectedErr: nil,
		},
		{
			name:        "client error",
			svcName:     "error-service",
			labels:      map[string]string{"app": "error"},
			selectorMap: map[string]string{"app": "error"},
			portsTCP:    []int{80},
			portsUDP:    []int{53},
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("create", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: k8s.ErrCreatingService.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			svc, err := suite.client.CreateService(context.Background(), tt.svcName, tt.labels, tt.selectorMap, tt.portsTCP, tt.portsUDP)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.Equal(suite.T(), tt.expectedErr.Error(), err.Error())
				return
			}

			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), tt.svcName, svc.Name)
		})
	}
}

func (suite *TestSuite) TestPatchService() {
	tests := []struct {
		name        string
		svcName     string
		labels      map[string]string
		selectorMap map[string]string
		portsTCP    []int
		portsUDP    []int
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name:        "successful patch",
			svcName:     "test-service",
			labels:      map[string]string{"app": "test"},
			selectorMap: map[string]string{"app": "test"},
			portsTCP:    []int{80},
			portsUDP:    []int{53},
			setupMock: func(clientset *fake.Clientset) {
				err := createService(clientset, "test-service", suite.namespace)
				require.NoError(suite.T(), err)
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
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("update", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: k8s.ErrPatchingService.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			svc, err := suite.client.PatchService(context.Background(), tt.svcName, tt.labels, tt.selectorMap, tt.portsTCP, tt.portsUDP)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.Equal(suite.T(), tt.expectedErr.Error(), err.Error())
				return
			}

			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), tt.svcName, svc.Name)
		})
	}
}

func (suite *TestSuite) TestDeleteService() {
	tests := []struct {
		name        string
		svcName     string
		setupMock   func(*fake.Clientset)
		expectedErr error
	}{
		{
			name:    "successful deletion",
			svcName: "test-service",
			setupMock: func(clientset *fake.Clientset) {
				err := createService(clientset, "test-service", suite.namespace)
				require.NoError(suite.T(), err)
			},
			expectedErr: nil,
		},
		{
			name:    "client error",
			svcName: "error-service",
			setupMock: func(clientset *fake.Clientset) {
				err := createService(clientset, "error-service", suite.namespace)
				require.NoError(suite.T(), err)

				clientset.PrependReactor("delete", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: k8s.ErrDeletingService.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			err := suite.client.DeleteService(context.Background(), tt.svcName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.Equal(suite.T(), tt.expectedErr.Error(), err.Error())
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

func (suite *TestSuite) TestGetServiceIP() {
	tests := []struct {
		name        string
		svcName     string
		setupMock   func(*fake.Clientset)
		expectedIP  string
		expectedErr error
	}{
		{
			name:    "successful retrieval",
			svcName: "test-service",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, &v1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-service",
							Namespace: suite.namespace,
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
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedIP:  "",
			expectedErr: k8s.ErrGettingService.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			ip, err := suite.client.GetServiceIP(context.Background(), tt.svcName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.Equal(suite.T(), tt.expectedErr.Error(), err.Error())
				return
			}

			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), tt.expectedIP, ip)
		})
	}
}

func (suite *TestSuite) TestWaitForService() {
	tests := []struct {
		name            string
		svcName         string
		setupMock       func(*fake.Clientset)
		serviceEndpoint string
		expectedErr     error
	}{
		{
			name:            "successful wait load balancer",
			svcName:         "test-service",
			serviceEndpoint: "127.0.0.1:8171",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, &v1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-service",
							Namespace: suite.namespace,
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
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, &v1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-service",
							Namespace: suite.namespace,
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
				clientset.PrependReactor("list", "nodes", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
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
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, &v1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-service",
							Namespace: suite.namespace,
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
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, &v1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "canceled-service",
							Namespace: suite.namespace,
						},
					}, nil
				})
			},
			expectedErr: k8s.ErrTimeoutWaitingForServiceReady,
		},
		{
			name:    "client error",
			svcName: "error-service",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedErr: k8s.ErrCheckingServiceReady.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.T().Parallel()
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			if tt.serviceEndpoint != "" {
				listener, err := startDummyServer(tt.serviceEndpoint)
				require.NoError(suite.T(), err)
				defer listener.Close()
			}

			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			err := suite.client.WaitForService(ctx, tt.svcName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.ErrorIs(suite.T(), err, tt.expectedErr)
				return
			}

			require.NoError(suite.T(), err)
		})
	}
}

// reset && go test -v ./pkg/k8s/ --run TestKubeManagerTestSuite/TestWaitForService

func (suite *TestSuite) TestGetServiceEndpoint() {
	tests := []struct {
		name        string
		svcName     string
		setupMock   func(*fake.Clientset)
		expectedEP  string
		expectedErr error
	}{
		{
			name:    "successful retrieval for ClusterIP",
			svcName: "test-service",
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, &v1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-service",
							Namespace: suite.namespace,
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
			setupMock: func(clientset *fake.Clientset) {
				clientset.PrependReactor("get", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("internal server error")
				})
			},
			expectedEP:  "",
			expectedErr: k8s.ErrGettingService.Wrap(errors.New("internal server error")),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.setupMock(suite.client.Clientset().(*fake.Clientset))

			ep, err := suite.client.GetServiceEndpoint(context.Background(), tt.svcName)
			if tt.expectedErr != nil {
				require.Error(suite.T(), err)
				assert.Equal(suite.T(), tt.expectedErr.Error(), err.Error())
				return
			}

			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), tt.expectedEP, ep)
		})
	}
}

func createService(clientset *fake.Clientset, name, namespace string) error {
	_, err := clientset.CoreV1().Services(namespace).Create(context.Background(), &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
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
