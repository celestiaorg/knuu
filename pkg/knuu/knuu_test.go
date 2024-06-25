package knuu

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/celestiaorg/knuu/pkg/builder/kaniko"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/minio"
)

const (
	testTimeout = 5 * time.Minute
)

type mockK8s struct {
	k8s.KubeManager
	mock.Mock
}

func (m *mockK8s) Clientset() kubernetes.Interface {
	return &kubernetes.Clientset{}
}

func (m *mockK8s) Namespace() string {
	return "test"
}

func (m *mockK8s) CreateServiceAccount(ctx context.Context, name string, labels map[string]string) error {
	return nil
}

func (m *mockK8s) CreateRole(ctx context.Context, name string, labels map[string]string, policyRules []rbacv1.PolicyRule) error {
	return nil
}

func (m *mockK8s) CreateRoleBinding(ctx context.Context, name string, labels map[string]string, role, serviceAccount string) error {
	return nil
}

func (m *mockK8s) CreateReplicaSet(ctx context.Context, rsConfig k8s.ReplicaSetConfig, init bool) (*appv1.ReplicaSet, error) {
	return &appv1.ReplicaSet{}, nil
}

func (m *mockK8s) IsReplicaSetRunning(ctx context.Context, name string) (bool, error) {
	return true, nil
}

func TestNew(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	tt := []struct {
		name          string
		options       Options
		expectedError error
		validateFunc  func(*testing.T, *Knuu)
	}{
		{
			name:          "Default initialization",
			options:       Options{},
			expectedError: nil,
			validateFunc: func(t *testing.T, k *Knuu) {
				assert.NotNil(t, k)
				assert.NotNil(t, k.Logger)
				assert.NotNil(t, k.K8sClient)
				assert.NotNil(t, k.ImageBuilder)
				assert.NotEmpty(t, k.TestScope)
				assert.Equal(t, defaultTimeout, k.timeout)
			},
		},
		{
			name:          "With Minio client without setting k8sClient",
			options:       Options{MinioClient: &minio.Minio{}},
			expectedError: ErrK8sClientNotSet,
		},
		{
			name: "With Minio client and K8sClient",
			options: Options{
				MinioClient: &minio.Minio{},
				K8sClient:   &mockK8s{},
			},
			expectedError: nil,
			validateFunc: func(t *testing.T, k *Knuu) {
				assert.NotNil(t, k)
				assert.NotNil(t, k.MinioClient)
				assert.NotNil(t, k.K8sClient)
			},
		},
		{
			name: "With custom Logger",
			options: Options{
				TestScope: "test",
				Logger:    &logrus.Logger{},
			},
			expectedError: nil,
			validateFunc: func(t *testing.T, k *Knuu) {
				assert.NotNil(t, k)
				assert.NotNil(t, k.Logger)
			},
		},
		{
			name: "With custom Timeout",
			options: Options{
				TestScope: "test",
				Timeout:   30 * time.Minute,
			},
			expectedError: nil,
			validateFunc: func(t *testing.T, k *Knuu) {
				assert.NotNil(t, k)
				assert.Equal(t, 30*time.Minute, k.timeout)
			},
		},
		{
			name: "With custom Image Builder",
			options: Options{
				TestScope:    "test",
				ImageBuilder: &kaniko.Kaniko{},
			},
			expectedError: nil,
			validateFunc: func(t *testing.T, k *Knuu) {
				assert.NotNil(t, k)
				assert.NotNil(t, k.ImageBuilder)
			},
		},
		{
			name: "With K8sClient but without TestScope",
			options: Options{
				K8sClient: &mockK8s{},
			},
			expectedError: nil,
			validateFunc: func(t *testing.T, k *Knuu) {
				assert.NotNil(t, k)
				assert.Equal(t, "test", k.TestScope)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			k, err := New(ctx, tc.options)
			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.expectedError)
				return
			}

			assert.NoError(t, err)
			tc.validateFunc(t, k)
		})
	}
}

func TestValidateOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     Options
		expectedErr error
	}{
		{
			name: "MinioClient set without K8sClient",
			options: Options{
				MinioClient: &minio.Minio{},
			},
			expectedErr: ErrK8sClientNotSet,
		},
		{
			name: "Both MinioClient and K8sClient set",
			options: Options{
				MinioClient: &minio.Minio{},
				K8sClient:   &mockK8s{},
			},
			expectedErr: nil,
		},
		{
			name: "TestScope and K8sClient not set",
			options: Options{
				TestScope: "",
				K8sClient: nil,
			},
			expectedErr: nil,
		},
		{
			name: "TestScope does not match K8sClient namespace",
			options: Options{
				TestScope: "another_scope",
				K8sClient: &mockK8s{},
			},
			expectedErr: ErrTestScopeMistMatch.WithParams("another_scope", "test"),
		},
		{
			name:        "No options set",
			options:     Options{},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOptions(tt.options)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}
