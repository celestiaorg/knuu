package instance

import (
	"context"
)

func (i *Instance) createTsharkCollectorInstance(ctx context.Context) (*Instance, error) {
	tsharkCollector, err := New("tshark-collector", i.SystemDependencies)
	if err != nil {
		return nil, err
	}
	if tsharkCollector.SetImage(ctx, "ttl.sh/68d00fc2-c488-4e3a-a787-5c0e8c8135d9:24h") != nil {
		return nil, err
	}
	if tsharkCollector.Commit() != nil {
		return nil, err
	}
	if tsharkCollector.SetCPU("100m") != nil {
		return nil, err
	}
	if tsharkCollector.SetMemory("250Mi", "250Mi") != nil {
		return nil, err
	}
	if tsharkCollector.AddVolume("/tshark", i.tsharkCollectorConfig.volumeSize) != nil {
		return nil, err
	}
	if tsharkCollector.SetEnvironmentVariable("STORAGE_ACCESS_KEY_ID", i.tsharkCollectorConfig.s3AccessKey) != nil {
		return nil, err
	}
	if tsharkCollector.SetEnvironmentVariable("STORAGE_SECRET_ACCESS_KEY", i.tsharkCollectorConfig.s3SecretKey) != nil {
		return nil, err
	}
	if tsharkCollector.SetEnvironmentVariable("STORAGE_REGION", i.tsharkCollectorConfig.s3Region) != nil {
		return nil, err
	}
	if tsharkCollector.SetEnvironmentVariable("STORAGE_BUCKET_NAME", i.tsharkCollectorConfig.s3Bucket) != nil {
		return nil, err
	}
	if tsharkCollector.SetEnvironmentVariable("STORAGE_KEY_PREFIX", i.tsharkCollectorConfig.s3KeyPrefix) != nil {
		return nil, err
	}
	if tsharkCollector.SetEnvironmentVariable("CAPTURE_FILE_NAME", i.k8sName+".pcapng") != nil {
		return nil, err
	}
	if tsharkCollector.AddCapability("NET_ADMIN") != nil {
		return nil, err
	}
	return tsharkCollector, nil
}
