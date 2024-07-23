package basic

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/celestiaorg/knuu/pkg/knuu"
	"github.com/celestiaorg/knuu/pkg/sidecars/bittwister"

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

	btSidecar := bittwister.New()
	require.NoError(t, main.AddSidecar(context.Background(), btSidecar))

	t.Cleanup(func() {
		if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
			t.Log("Skipping cleanup")
			return
		}

		require.NoError(t, main.Destroy(), "Error destroying instance")
	})

	require.NoError(t, main.Start(), "Error starting main instance")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	require.NoError(t, btSidecar.WaitForStart(ctx), "Error waiting for BitTwister to start")

	// test if BitTwister running in a sidecar is accessible
	err = btSidecar.SetBandwidthLimit(1000)
	assert.NoError(t, err, "Error setting bandwidth limit")

	// Check if the BitTwister service is set
	out, err := btSidecar.AllServicesStatus()
	assert.NoError(t, err, "Error getting all services status")
	assert.GreaterOrEqual(t, len(out), 1, "No services found")
	assert.NotEmpty(t, out[0].Name, "Service name is empty")
}

func TestAddHostWithReadyCheck(t *testing.T) {
	t.Parallel()

	target, err := knuu.NewInstance("target")
	require.NoError(t, err, "Error creating instance 'target'")

	err = target.SetImage("nginx:latest")
	require.NoError(t, err, "error setting image")

	require.NoError(t, target.Commit(), "error committing instance")

	t.Cleanup(func() {
		if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
			t.Log("Skipping cleanup")
			return
		}
		if err := target.Destroy(); err != nil {
			t.Logf("error destroying instance: %v", err)
		}
	})

	const port = 80
	require.NoError(t, target.AddPortTCP(port), "error adding port")
	require.NoError(t, target.Start(), "error starting instance")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// checkFunc verifies the proxy is serving the nginx page
	checkFunc := func(host string) (bool, error) {
		resp, err := http.Get(host)
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return false, nil
		}
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}
		return strings.Contains(string(bodyBytes), "Welcome to nginx!"), nil
	}

	host, err := target.AddHostWithReadyCheck(ctx, port, checkFunc)
	require.NoError(t, err, "error adding host with ready check")
	assert.NotEmpty(t, host, "host should not be empty")

	// Additional verification that the host is accessible
	ok, err := checkFunc(host)
	require.NoError(t, err, "error checking host")
	assert.True(t, ok, "Host should be ready and serving content: expected true, got false")
}
