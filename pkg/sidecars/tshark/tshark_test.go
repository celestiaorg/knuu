package tshark

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	discfake "k8s.io/client-go/discovery/fake"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/system"
)

func TestTsharkInitialize(t *testing.T) {
	tests := []struct {
		name    string
		config  *Tshark
		wantErr error
	}{
		{
			name: "Valid configuration",
			config: &Tshark{
				VolumeSize:     resource.MustParse("1Gi"),
				S3AccessKey:    "testAccessKey",
				S3SecretKey:    "testSecretKey",
				S3Region:       "us-west-1",
				S3Bucket:       "testBucket",
				CreateBucket:   true,
				S3KeyPrefix:    "testPrefix",
				S3Endpoint:     "http://localhost:9000",
				UploadInterval: time.Minute * 5,
			},
			wantErr: nil,
		},
		{
			name: "Invalid configuration - empty S3Region",
			config: &Tshark{
				VolumeSize:     resource.MustParse("1Gi"),
				S3AccessKey:    "testAccessKey",
				S3SecretKey:    "testSecretKey",
				S3Bucket:       "testBucket",
				CreateBucket:   true,
				S3KeyPrefix:    "testPrefix",
				S3Endpoint:     "http://localhost:9000",
				UploadInterval: time.Minute * 5,
			},
			wantErr: ErrTsharkCollectorS3RegionOrBucketEmpty,
		},
		{
			name: "Invalid configuration - zero VolumeSize",
			config: &Tshark{
				VolumeSize:     resource.MustParse("0"),
				S3AccessKey:    "testAccessKey",
				S3SecretKey:    "testSecretKey",
				S3Region:       "us-west-1",
				S3Bucket:       "testBucket",
				CreateBucket:   true,
				S3KeyPrefix:    "testPrefix",
				S3Endpoint:     "http://localhost:9000",
				UploadInterval: time.Minute * 5,
			},
			wantErr: ErrTsharkCollectorInvalidVolumeSize.WithParams("0"),
		},
	}

	ctx := context.Background()
	logger := logrus.New()
	k8sClient, err := k8s.NewClientCustom(
		context.Background(),
		fake.NewSimpleClientset(),
		&discfake.FakeDiscovery{Fake: &k8stesting.Fake{}},
		dynfake.NewSimpleDynamicClient(runtime.NewScheme()),
		"testNamespace",
		logrus.New(),
	)
	require.NoError(t, err)
	sysDeps := &system.SystemDependencies{
		K8sClient: k8sClient,
		Logger:    logger,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := tt.config.Initialize(ctx, sysDeps)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTsharkPreStart(t *testing.T) {
	tshark := &Tshark{}
	err := tshark.PreStart(context.Background())
	assert.Error(t, err)
	assert.Equal(t, ErrTsharkCollectorNotInitialized, err)
}

func TestTsharkValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Tshark
		wantErr error
	}{
		{
			name: "Valid configuration",
			config: &Tshark{
				VolumeSize: resource.MustParse("1Gi"),
				S3Region:   "us-west-1",
				S3Bucket:   "testBucket",
			},
			wantErr: nil,
		},
		{
			name: "Invalid configuration - zero VolumeSize",
			config: &Tshark{
				VolumeSize: resource.MustParse("0"),
				S3Region:   "us-west-1",
				S3Bucket:   "testBucket",
			},
			wantErr: ErrTsharkCollectorInvalidVolumeSize.WithParams("0"),
		},
		{
			name: "Invalid configuration - empty S3Region",
			config: &Tshark{
				VolumeSize: resource.MustParse("1Gi"),
				S3Region:   "",
				S3Bucket:   "testBucket",
			},
			wantErr: ErrTsharkCollectorS3RegionOrBucketEmpty.WithParams("", "testBucket"),
		},
		{
			name: "Invalid configuration - empty S3Bucket",
			config: &Tshark{
				VolumeSize: resource.MustParse("1Gi"),
				S3Region:   "us-west-1",
				S3Bucket:   "",
			},
			wantErr: ErrTsharkCollectorS3RegionOrBucketEmpty.WithParams("us-west-1", ""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validateConfig()

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTsharkCloneWithSuffix(t *testing.T) {
	testInstance, err := instance.New("testInstance", &system.SystemDependencies{})
	require.NoError(t, err)

	tshark := &Tshark{
		VolumeSize:     resource.MustParse("1Gi"),
		S3AccessKey:    "testAccessKey",
		S3SecretKey:    "testSecretKey",
		S3Region:       "us-west-1",
		S3Bucket:       "testBucket",
		CreateBucket:   true,
		S3KeyPrefix:    "testPrefix",
		S3Endpoint:     "http://localhost:9000",
		UploadInterval: time.Minute * 5,
		instance:       testInstance,
	}

	clone, err := tshark.Clone()
	require.NoError(t, err)

	assert.Equal(t, tshark.VolumeSize, clone.(*Tshark).VolumeSize)
	assert.Equal(t, tshark.S3AccessKey, clone.(*Tshark).S3AccessKey)
	assert.Equal(t, tshark.S3SecretKey, clone.(*Tshark).S3SecretKey)
	assert.Equal(t, tshark.S3Region, clone.(*Tshark).S3Region)
	assert.Equal(t, tshark.S3Bucket, clone.(*Tshark).S3Bucket)
	assert.Equal(t, tshark.CreateBucket, clone.(*Tshark).CreateBucket)
	assert.Equal(t, tshark.S3KeyPrefix, clone.(*Tshark).S3KeyPrefix)
	assert.Equal(t, tshark.S3Endpoint, clone.(*Tshark).S3Endpoint)
	assert.Equal(t, tshark.UploadInterval, clone.(*Tshark).UploadInterval)
	assert.NotEmpty(t, clone.(*Tshark).instance)
}
