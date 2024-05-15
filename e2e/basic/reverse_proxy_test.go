package basic

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"testing"
	"time"

	"github.com/celestiaorg/knuu/pkg/knuu"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	// Prepare iperf client & server

	require.NoError(t, main.EnableBitTwister(), "Error enabling BitTwister")
	require.NoError(t, main.Start(), "Error starting main instance")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	{
		out, err := main.BitTwister.Client().AllServicesStatus()
		fmt.Printf("err: %v\n", err)

		// wait for Ctrl+C signal for 10 minutes
		// Setup a channel to listen for the interrupt signal (Ctrl+C)
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)

		// Create a timeout to stop waiting after 10 minutes
		timeout := time.After(30 * time.Minute)

		select {
		case <-sigChan:
			fmt.Println("Ctrl+C received, stopping the test.")
		case <-timeout:
			fmt.Println("30 minutes passed without Ctrl+C, continuing the test.")
		}

		require.NoError(t, err, "Error getting all services status")
		assert.NotEmpty(t, out, "No services found")
	}
	require.NoError(t, main.BitTwister.WaitForStart(ctx), "Error waiting for BitTwister to start")

	// test if BitTwister running in a sidecar is accessible

	err = main.SetBandwidthLimit(1000)
	assert.NoError(t, err, "Error setting bandwidth limit")

	// Check if the BitTwister service is set
	out, err := main.BitTwister.Client().AllServicesStatus()
	require.NoError(t, err, "Error getting all services status")
	assert.NotEmpty(t, out, "No services found")

	for _, svc := range out {
		fmt.Printf("\nsvc: %#v\n", svc)
	}
}

// reset && LOG_LEVEL=debug go test -v ./e2e/basic/ --run TestReverseProxy -timeout 60m
