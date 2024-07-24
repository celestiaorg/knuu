package system

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/celestiaorg/knuu/pkg/knuu"
)

const (
	callbackName         = "callback-test"
	sleepTimeBeforeReady = "1" // second
)

func TestStartWithCallback(t *testing.T) {
	t.Parallel()

	// Setup
	ctx := context.Background()

	// The default image builder is kaniko here
	kn, err := knuu.New(ctx, knuu.Options{})
	require.NoError(t, err, "Error creating knuu")

	target, err := kn.NewInstance(callbackName)
	require.NoError(t, err, "Error creating instance")

	require.NoError(t, target.SetImage(ctx, nginxImage))

	// This probe is used to make sure the instance will not be ready for a second so the
	// second execute command must fail and the first one with callback must succeed
	err = target.SetReadinessProbe(&corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/",
				Port: intstr.FromInt(nginxPort),
			},
		},
	})
	require.NoError(t, err, "Error setting readiness probe")

	err = target.SetCommand([]string{"sleep", sleepTimeBeforeReady, "&&", nginxCommand}...)
	require.NoError(t, err, "Error setting command")

	t.Cleanup(func() {
		if err := kn.CleanUp(ctx); err != nil {
			t.Logf("Error cleaning up knuu: %v", err)
		}
	})

	require.NoError(t, target.Commit())

	wg := sync.WaitGroup{}
	require.NoError(t, target.StartWithCallback(ctx, func() {
		wg.Add(1)
		defer wg.Done()
		// This should Not fail as the instance will be ready when this is called
		out, err := target.ExecuteCommand(ctx, "curl", "-s", "http://localhost")
		assert.NoError(t, err, "Error executing command")
		assert.Contains(t, out, "Welcome to nginx")
	}))

	// This should fail as the instance is not ready yet
	out, err := target.ExecuteCommand(ctx, "curl", "-s", "http://localhost")
	assert.Error(t, err, "Error executing command")
	assert.Empty(t, out, "Output should be empty")

	// We need to have this to allow the async callback to finish
	wg.Wait()
}
