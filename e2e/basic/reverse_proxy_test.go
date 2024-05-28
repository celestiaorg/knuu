package basic

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/celestiaorg/knuu/pkg/knuu"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReverseProxy is a test function that verifies the functionality of a reverse proxy setup.
// It mainly tests the ability to reach to a service running in a sidecar like BitTwister.
// It calls an endpoint of the service and checks if the response is as expected.
func TestReverseProxy(t *testing.T) {
	t.Parallel()
	// Setup

	main, err := knuu.NewInstance("main")
	require.NoError(t, err, "Error creating instance")

	err = main.SetImage("alpine:latest")
	require.NoError(t, err, "Error setting image")

	err = main.SetCommand("sleep", "infinite")
	require.NoError(t, err, "Error executing command")

	require.NoError(t, main.Commit(), "Error committing instance")

	t.Cleanup(func() {
		if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
			t.Log("Skipping cleanup")
			return
		}

		require.NoError(t, main.Destroy(), "Error destroying instance")
	})

	require.NoError(t, main.EnableBitTwister(), "Error enabling BitTwister")
	require.NoError(t, main.Start(), "Error starting main instance")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	require.NoError(t, main.BitTwister.WaitForStart(ctx), "Error waiting for BitTwister to start")

	// test if BitTwister running in a sidecar is accessible
	err = main.SetBandwidthLimit(1000)
	assert.NoError(t, err, "Error setting bandwidth limit")

	// Check if the BitTwister service is set
	out, err := main.BitTwister.Client().AllServicesStatus()
	assert.NoError(t, err, "Error getting all services status")
	assert.GreaterOrEqual(t, len(out), 1, "No services found")
	assert.NotEmpty(t, out[0].Name, "Service name is empty")
}
