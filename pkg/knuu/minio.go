package knuu

import (
	"context"
	"io"
)

const minioBucketName = "knuu"

func (k *Knuu) initMinio(ctx context.Context) error {
	if k.MinioCli == nil {
		return ErrMinioNotInitialized
	}

	ok, err := k.MinioCli.IsMinioDeployed(ctx)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return k.MinioCli.DeployMinio(ctx)
}

// contentName is a unique string to identify the content in Minio
func (k *Knuu) PushFileToMinio(ctx context.Context, contentName string, reader io.Reader) error {
	if err := k.initMinio(ctx); err != nil {
		return err
	}
	return k.MinioCli.PushToMinio(ctx, reader, contentName, minioBucketName)
}

func (k *Knuu) GetMinioURL(ctx context.Context, contentName string) (string, error) {
	if err := k.initMinio(ctx); err != nil {
		return "", err
	}
	return k.MinioCli.GetMinioURL(ctx, contentName, minioBucketName)
}
