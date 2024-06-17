package knuu

import (
	"context"
	"io"

	"github.com/celestiaorg/knuu/pkg/minio"
)

const minioBucketName = "knuu"

// initMinio initializes the Minio client
// Since not always we need minio to be deployed and ready,
// We deploy it on the first use. i.e. minio.New deploys it
func (k *Knuu) initMinio(ctx context.Context) error {
	if k.MinioClient != nil {
		return nil
	}

	var err error
	k.MinioClient, err = minio.New(ctx, k.K8sClient)
	return err
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
