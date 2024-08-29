package tshark

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/system"
)

const (
	DefaultImage = "ghcr.io/celestiaorg/tshark-s3:pr-11"

	tsharkCollectorName        = "tshark-collector"
	tsharkCollectorVolumePath  = "/tshark"
	netAdminCapability         = "NET_ADMIN"
	TsharkCaptureFileExtension = ".pcapng"

	envStorageAccessKeyID     = "STORAGE_ACCESS_KEY_ID"
	envStorageSecretAccessKey = "STORAGE_SECRET_ACCESS_KEY"
	envStorageRegion          = "STORAGE_REGION"
	envStorageBucketName      = "STORAGE_BUCKET_NAME"
	envCreateBucket           = "STORAGE_CREATE_BUCKET"
	envStorageKeyPrefix       = "STORAGE_KEY_PREFIX"
	envStorageEndpoint        = "STORAGE_ENDPOINT"
	envCaptureFileName        = "CAPTURE_FILE_NAME"
	envUploadInterval         = "UPLOAD_INTERVAL"
)

// Tshark represents the configuration for the tshark collector
type Tshark struct {
	instance *instance.Instance
	Image    string
	// VolumeSize is the size of the volume to use for the tshark collector
	VolumeSize resource.Quantity
	// S3AccessKey is the access key to use for the s3 server
	S3AccessKey string
	// S3SecretKey is the secret key to use for the s3 server
	S3SecretKey string
	// S3Region is the region of the s3 server
	S3Region string
	// S3Bucket is the bucket to use for the s3 server
	S3Bucket string
	// CreateBucket is the flag to create the bucket if it does not exist
	CreateBucket bool
	// S3KeyPrefix is the key prefix to use for the s3 server
	S3KeyPrefix string
	// S3Endpoint is the endpoint of the s3 server
	S3Endpoint string

	// UploadInterval is the interval at which the tshark collector will upload the pcap file to the s3 server
	UploadInterval time.Duration
}

var _ instance.SidecarManager = (*Tshark)(nil)

var (
	tsharkCollectorCPU    = resource.MustParse("100m")
	tsharkCollectorMemory = resource.MustParse("250Mi")
)

// Initialize initializes the BitTwister sidecar
// and it is called once the instance.AddSidecar is called
func (t *Tshark) Initialize(ctx context.Context, sysDeps *system.SystemDependencies) error {
	if err := t.validateConfig(); err != nil {
		return err
	}

	if t.Image == "" {
		t.Image = DefaultImage
	}

	var err error
	t.instance, err = instance.New(tsharkCollectorName, sysDeps)
	if err != nil {
		return ErrCreatingTsharkCollectorInstance.Wrap(err)
	}
	t.instance.Sidecars().SetIsSidecar(true)

	if err := t.instance.Build().SetImage(ctx, t.Image); err != nil {
		return ErrSettingTsharkCollectorImage.Wrap(err)
	}

	if err := t.instance.Build().Commit(ctx); err != nil {
		return ErrCommittingTsharkCollectorInstance.Wrap(err)
	}

	if err := t.instance.Resources().SetCPU(tsharkCollectorCPU); err != nil {
		return ErrSettingTsharkCollectorCPU.Wrap(err)
	}

	if err := t.instance.Resources().SetMemory(tsharkCollectorMemory, tsharkCollectorMemory); err != nil {
		return ErrSettingTsharkCollectorMemory.Wrap(err)
	}
	if err := t.instance.Storage().AddVolume(tsharkCollectorVolumePath, t.VolumeSize); err != nil {
		return ErrAddingTsharkCollectorVolume.Wrap(err)
	}

	envVars := map[string]string{
		envStorageAccessKeyID:     t.S3AccessKey,
		envStorageSecretAccessKey: t.S3SecretKey,
		envStorageRegion:          t.S3Region,
		envStorageBucketName:      t.S3Bucket,
		envStorageKeyPrefix:       t.S3KeyPrefix,
		envCaptureFileName:        t.instance.Name() + TsharkCaptureFileExtension,
		envStorageEndpoint:        t.S3Endpoint,
		envUploadInterval:         fmt.Sprintf("%d", int64(t.UploadInterval.Seconds())),
		envCreateBucket:           fmt.Sprintf("%t", t.CreateBucket),
	}

	for key, value := range envVars {
		if err := t.instance.Build().SetEnvironmentVariable(key, value); err != nil {
			return ErrSettingTsharkCollectorEnv.Wrap(err)
		}
	}
	if err := t.instance.Security().AddKubernetesCapability(netAdminCapability); err != nil {
		return ErrAddingTsharkCollectorCapability.Wrap(err)
	}
	return nil
}

// PreStart is called before the instance is started
// It is used to prepare the sidecar for the instance to start
func (t *Tshark) PreStart(ctx context.Context) error {
	if t.instance == nil {
		return ErrTsharkCollectorNotInitialized
	}
	return nil
}

func (t *Tshark) Instance() *instance.Instance {
	return t.instance
}

func (t *Tshark) Clone() (instance.SidecarManager, error) {
	clone, err := t.instance.CloneWithName(tsharkCollectorName)
	if err != nil {
		return nil, err
	}
	return &Tshark{
		instance:       clone,
		VolumeSize:     t.VolumeSize,
		S3AccessKey:    t.S3AccessKey,
		S3SecretKey:    t.S3SecretKey,
		S3Region:       t.S3Region,
		S3Bucket:       t.S3Bucket,
		CreateBucket:   t.CreateBucket,
		S3KeyPrefix:    t.S3KeyPrefix,
		S3Endpoint:     t.S3Endpoint,
		UploadInterval: t.UploadInterval,
	}, nil
}
