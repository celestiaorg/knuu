package instance

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	tsharkCollectorName        = "tshark-collector"
	tsharkCollectorImage       = "ghcr.io/celestiaorg/tshark-s3:pr-11"
	tsharkCollectorCPU         = "100m"
	tsharkCollectorMemory      = "250Mi"
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

func (i *Instance) createTsharkCollectorInstance(ctx context.Context) (*Instance, error) {
	if i.tsharkCollectorConfig == nil {
		return nil, ErrTsharkCollectorConfigNotSet
	}

	tsc, err := New(tsharkCollectorName, i.SystemDependencies)
	if err != nil {
		return nil, err
	}
	if err := tsc.SetImage(ctx, tsharkCollectorImage); err != nil {
		return nil, err
	}
	if err := tsc.Commit(); err != nil {
		return nil, err
	}
	if err := tsc.SetCPU(resource.MustParse(tsharkCollectorCPU)); err != nil {
		return nil, err
	}
	err = tsc.SetMemory(
		resource.MustParse(tsharkCollectorMemory),
		resource.MustParse(tsharkCollectorMemory),
	)
	if err != nil {
		return nil, err
	}
	err = tsc.AddVolume(
		tsharkCollectorVolumePath,
		resource.MustParse(i.tsharkCollectorConfig.VolumeSize),
	)
	if err != nil {
		return nil, err
	}

	envVars := map[string]string{
		envStorageAccessKeyID:     i.tsharkCollectorConfig.S3AccessKey,
		envStorageSecretAccessKey: i.tsharkCollectorConfig.S3SecretKey,
		envStorageRegion:          i.tsharkCollectorConfig.S3Region,
		envStorageBucketName:      i.tsharkCollectorConfig.S3Bucket,
		envStorageKeyPrefix:       i.tsharkCollectorConfig.S3KeyPrefix,
		envCaptureFileName:        i.k8sName + TsharkCaptureFileExtension,
		envStorageEndpoint:        i.tsharkCollectorConfig.S3Endpoint,
		envUploadInterval:         fmt.Sprintf("%d", int64(i.tsharkCollectorConfig.UploadInterval.Seconds())),
		envCreateBucket:           fmt.Sprintf("%t", i.tsharkCollectorConfig.CreateBucket),
	}

	for key, value := range envVars {
		if err := tsc.SetEnvironmentVariable(key, value); err != nil {
			return nil, err
		}
	}
	if err := tsc.AddCapability(netAdminCapability); err != nil {
		return nil, err
	}
	return tsc, nil
}
