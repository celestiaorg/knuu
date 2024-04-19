package knuu

import (
	"context"
	"fmt"
	"io"

	"github.com/celestiaorg/knuu/pkg/minio"
)

const minioBucketName = "knuu"

var minioClient *minio.Minio

func initMinio(ctx context.Context) error {
	if minioClient == nil {
		return fmt.Errorf("minio not initialized")
	}

	ok, err := minioClient.IsMinioDeployed(ctx)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return minioClient.DeployMinio(ctx)
}

// contentName is a unique string to identify the content in Minio
func PushFileToMinio(ctx context.Context, contentName string, reader io.Reader) error {
	if err := initMinio(ctx); err != nil {
		return err
	}
	return minioClient.PushToMinio(ctx, reader, contentName, minioBucketName)
}

func GetMinioURL(ctx context.Context, contentName string) (string, error) {
	if err := initMinio(ctx); err != nil {
		return "", err
	}
	return minioClient.GetMinioURL(ctx, contentName, minioBucketName)
}
