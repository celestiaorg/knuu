package basic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/knuu/pkg/knuu"
	"github.com/celestiaorg/knuu/pkg/sidecars/tshark"
)

const (
	s3BucketName = "tshark-test-bucket"
	s3Location   = "eu-east-1"
)

func TestTshark(t *testing.T) {
	t.Parallel()
	// Setup

	ctx := context.Background()

	kn, err := knuu.New(ctx, knuu.Options{})
	require.NoError(t, err, "error creating knuu")

	defer func() {
		return
		if err := kn.CleanUp(ctx); err != nil {
			t.Logf("error cleaning up knuu: %v", err)
		}
	}()

	scope := kn.Scope()
	t.Logf("Test scope: %s", scope)

	target, err := kn.NewInstance("busybox")
	require.NoError(t, err, "error creating instance")

	err = target.SetImage(ctx, "busybox")
	require.NoError(t, err, "error setting image")

	err = target.SetCommand("sleep", "infinity")
	require.NoError(t, err, "error setting command")

	t.Log("deploying minio as s3 backend")
	err = kn.MinioClient.DeployMinio(ctx)
	require.NoError(t, err, "error deploying minio")

	t.Log("getting minio configs")
	minioConf, err := kn.MinioClient.GetConfigs(ctx)
	require.NoError(t, err, "error getting S3 (minio) configs")

	var (
		filename  = target.K8sName() + tshark.TsharkCaptureFileExtension
		keyPrefix = "tshark/" + scope
		fileKey   = filepath.Join(keyPrefix, filename)
	)

	fmt.Printf("fileKey: %v\n", fileKey)

	tsc := &tshark.Tshark{
		VolumeSize:     "10Gi",
		S3AccessKey:    minioConf.AccessKeyID,
		S3SecretKey:    minioConf.SecretAccessKey,
		S3Region:       s3Location,
		S3Bucket:       s3BucketName,
		CreateBucket:   true, // Since we fire up a fresh minio server, we need to create the bucket
		S3KeyPrefix:    keyPrefix,
		S3Endpoint:     minioConf.Endpoint,
		UploadInterval: 1 * time.Second, // for sake of the test we keep this short
	}

	err = target.AddSidecar(ctx, tsc)
	require.NoError(t, err, "error adding tshark collector")

	err = target.Commit()
	require.NoError(t, err, "error committing instance")

	// Test logic

	t.Log("starting target instance")
	err = target.Start(ctx)
	require.NoError(t, err, "error starting instance")

	err = target.WaitInstanceIsRunning(ctx)
	require.NoError(t, err, "error waiting for instance to be running")

	// Perform a ping to do generate network traffic to allow tshark to capture it
	_, err = target.ExecuteCommand(ctx, "ping", "-c", "4", "google.com")
	require.NoError(t, err, "error executing command")

	url, err := kn.MinioClient.GetMinioURL(ctx, fileKey, s3BucketName)
	require.NoError(t, err, "error getting minio url")

	fmt.Printf("url: %v\n", url)

	resp, err := http.Get(url)
	require.NoError(t, err, "error downloading from minio URL")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "URL does not exist or is not accessible")

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "error reading response body")
	assert.NotEmpty(t, bodyBytes, "downloaded log file is empty")
}
