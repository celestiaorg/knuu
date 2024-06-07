package instance

import (
	"context"

	"github.com/celestiaorg/knuu/pkg/knuu"
)

const (
	tsharkCollectorName       = "tshark-collector"
	tsharkCollectorImage      = "ghcr.io/celestiaorg/tshark-s3:pr-5"
	tsharkCollectorCPU        = "100m"
	tsharkCollectorMemory     = "250Mi"
	tsharkCollectorVolumePath = "/tshark"
	netAdminCapability        = "NET_ADMIN"
)

func (i *Instance) createTsharkCollectorInstance(ctx context.Context) (*Instance, error) {
	tsharkCollector, err := New(tsharkCollectorName, i.SystemDependencies)
	if err != nil {
		return nil, err
	}
	if tsharkCollector.SetImage(ctx, tsharkCollectorImage) != nil {
		return nil, err
	}
	if tsharkCollector.Commit() != nil {
		return nil, err
	}
	if tsharkCollector.SetCPU(tsharkCollectorCPU) != nil {
		return nil, err
	}
	if tsharkCollector.SetMemory(tsharkCollectorMemory, tsharkCollectorMemory) != nil {
		return nil, err
	}
	if tsharkCollector.AddVolume(tsharkCollectorVolumePath, i.tsharkCollectorConfig.volumeSize) != nil {
		return nil, err
	}
	envVars := map[string]string{
		"STORAGE_ACCESS_KEY_ID":     i.tsharkCollectorConfig.s3AccessKey,
		"STORAGE_SECRET_ACCESS_KEY": i.tsharkCollectorConfig.s3SecretKey,
		"STORAGE_REGION":            i.tsharkCollectorConfig.s3Region,
		"STORAGE_BUCKET_NAME":       i.tsharkCollectorConfig.s3Bucket,
		"STORAGE_KEY_PREFIX":        i.tsharkCollectorConfig.s3KeyPrefix + "/" + knuu.Scope(),
		"CAPTURE_FILE_NAME":         i.k8sName + ".pcapng",
	}

	for key, value := range envVars {
		if tsharkCollector.SetEnvironmentVariable(key, value) != nil {
			return nil, err
		}
	}
	if tsharkCollector.AddCapability(netAdminCapability) != nil {
		return nil, err
	}
	return tsharkCollector, nil
}
