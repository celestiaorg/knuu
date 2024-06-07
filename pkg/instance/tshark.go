package instance

import (
	"context"
	"fmt"
)

const (
	tsharkCollectorName        = "tshark-collector"
	tsharkCollectorImage       = "ghcr.io/celestiaorg/tshark-s3:f35863a"
	tsharkCollectorCPU         = "100m"
	tsharkCollectorMemory      = "250Mi"
	tsharkCollectorVolumePath  = "/tshark"
	netAdminCapability         = "NET_ADMIN"
	TsharkCaptureFileExtension = ".pcapng"

	envStorageAccessKeyID     = "STORAGE_ACCESS_KEY_ID"
	envStorageSecretAccessKey = "STORAGE_SECRET_ACCESS_KEY"
	envStorageRegion          = "STORAGE_REGION"
	envStorageBucketName      = "STORAGE_BUCKET_NAME"
	envStorageKeyPrefix       = "STORAGE_KEY_PREFIX"
	envStorageEndpoint        = "STORAGE_ENDPOINT"
	envCaptureFileName        = "CAPTURE_FILE_NAME"
	envUploadInterval         = "UPLOAD_INTERVAL"
)

func (i *Instance) createTsharkCollectorInstance(ctx context.Context) (*Instance, error) {
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
	if err := tsc.SetCPU(tsharkCollectorCPU); err != nil {
		return nil, err
	}
	if err := tsc.SetMemory(tsharkCollectorMemory, tsharkCollectorMemory); err != nil {
		return nil, err
	}
	if err := tsc.AddVolume(tsharkCollectorVolumePath, i.tsharkCollectorConfig.VolumeSize); err != nil {
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
