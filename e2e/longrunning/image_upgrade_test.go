package longrunning

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestImageUpgrade(t *testing.T) {
	const (
		instanceName = "image-upgrade"
		firstImage   = "alpine:3.20.3"
		secondImage  = "nginx:latest"
		volumeSize   = "1Gi"
		volumePath   = "/tmp/test-id"
		fileContent  = "hello world"
		filePath     = volumePath + "/test-id.txt"
	)

	ctx := context.Background()

	ins1, err := createInstance(ctx, instanceName, "", firstImage)
	require.NoError(t, err)

	err = ins1.Storage().AddVolume(volumePath, resource.MustParse(volumeSize))
	require.NoError(t, err)

	require.NoError(t, ins1.Execution().Start(ctx))

	testScope := ins1.Scope
	t.Logf("Scope: %s", testScope)

	_, err = ins1.Execution().ExecuteCommand(ctx, "echo", fileContent, ">", filePath)
	require.NoError(t, err)

	t.Logf("Waiting for 5 seconds to simulate a long running process")
	time.Sleep(5 * time.Second)

	ins2, err := createInstance(ctx, instanceName, testScope, firstImage)
	require.NoError(t, err)

	err = ins2.Storage().AddVolume(volumePath, resource.MustParse(volumeSize))
	require.NoError(t, err)

	require.NoError(t, ins2.Execution().Start(ctx))

	// To upgrade the image, first the instance must be stopped
	require.NoError(t, ins2.Execution().Stop(ctx))

	// Now we can upgrade the image
	require.NoError(t, ins2.Build().SetImage(ctx, secondImage))
	require.NoError(t, ins2.Build().Commit(ctx))
	require.NoError(t, ins2.Execution().Start(ctx))

	// Test if the alpine image is replaced with the nginx image successfully
	out, err := ins2.Execution().ExecuteCommand(ctx, "cat", "/etc/nginx/nginx.conf")
	require.NoError(t, err)
	require.NotEmpty(t, out)

	// Test if the volume is persisted across the image upgrade
	out, err = ins2.Execution().ExecuteCommand(ctx, "cat", filePath)
	require.NoError(t, err)
	require.Contains(t, out, fileContent)
}
