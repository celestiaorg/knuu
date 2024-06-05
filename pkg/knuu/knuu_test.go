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

func (m *mockK8s) Clientset() *kubernetes.Clientset {
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
		name         string
		options      []Option
		expectError  bool
		validateFunc func(*testing.T, *Knuu)
	}{
		{
			name:        "Default initialization",
			options:     nil,
			expectError: false,
			validateFunc: func(t *testing.T, k *Knuu) {
				assert.NotNil(t, k)
				assert.NotNil(t, k.Logger)
				assert.NotNil(t, k.K8sCli)
				assert.NotNil(t, k.MinioCli)
				assert.NotNil(t, k.ImageBuilder)
				assert.Equal(t, defaultTimeout, k.timeout)
			},
		},
		{
			name: "With custom Logger",
			options: []Option{
				WithLogger(&logrus.Logger{}),
			},
			expectError: false,
			validateFunc: func(t *testing.T, k *Knuu) {
				assert.NotNil(t, k)
				assert.NotNil(t, k.Logger)
			},
		},
		{
			name: "With custom Timeout",
			options: []Option{
				WithTimeout(30 * time.Minute),
			},
			expectError: false,
			validateFunc: func(t *testing.T, k *Knuu) {
				assert.NotNil(t, k)
				assert.Equal(t, 30*time.Minute, k.timeout)
			},
		},
		{
			name: "With custom K8s client",
			options: []Option{
				WithK8s(&mockK8s{}),
			},
			expectError: false,
			validateFunc: func(t *testing.T, k *Knuu) {
				assert.NotNil(t, k)
				assert.NotNil(t, k.K8sCli)
			},
		},
		{
			name: "With custom Minio client",
			options: []Option{
				WithMinio(&minio.Minio{}),
			},
			expectError: false,
			validateFunc: func(t *testing.T, k *Knuu) {
				assert.NotNil(t, k)
				assert.NotNil(t, k.MinioCli)
			},
		},
		{
			name: "With custom Image Builder",
			options: []Option{
				WithImageBuilder(&kaniko.Kaniko{}),
			},
			expectError: false,
			validateFunc: func(t *testing.T, k *Knuu) {
				assert.NotNil(t, k)
				assert.NotNil(t, k.ImageBuilder)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			k, err := New(ctx, tc.options...)
			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			tc.validateFunc(t, k)
		})
	}
}
