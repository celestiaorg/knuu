package knuu

import (
	"context"
	"io"
)

const minioBucketName = "knuu"

func (k *Knuu) initMinio(ctx context.Context) error {
	if k.MinioClient == nil {
		return ErrMinioNotInitialized
	}

	ok, err := k.MinioClient.IsMinioDeployed(ctx)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return k.MinioClient.DeployMinio(ctx)
}

// contentName is a unique string to identify the content in Minio
func (k *Knuu) PushFileToMinio(ctx context.Context, contentName string, reader io.Reader) error {
	if err := k.initMinio(ctx); err != nil {
		return err
	}
	return k.MinioClient.PushToMinio(ctx, reader, contentName, minioBucketName)
}

func (k *Knuu) GetMinioURL(ctx context.Context, contentName string) (string, error) {
	if err := k.initMinio(ctx); err != nil {
		return "", err
	}
	return k.MinioClient.GetMinioURL(ctx, contentName, minioBucketName)
}
