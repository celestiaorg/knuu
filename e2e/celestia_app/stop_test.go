package celestia_app

import (
	"testing"
	"time"

	"github.com/celestiaorg/knuu/e2e/celestia_app/utils"
	"github.com/celestiaorg/knuu/pkg/knuu"
	"github.com/stretchr/testify/require"
)

func TestStop(t *testing.T) {
	t.Parallel()
	// Setup

	executor, err := knuu.NewExecutor()
	if err != nil {
		t.Fatalf("Error creating executor: %v", err)
	}

	validator, err := Instances["validator"].Clone()
	if err != nil {
		t.Fatalf("Error cloning instance: %v", err)
	}
	full, err := Instances["full"].Clone()
	if err != nil {
		t.Fatalf("Error cloning instance: %v", err)
	}
	err = full.AddVolumeWithOwner("/home/celestia", "1Gi", 10001)
	if err != nil {
		t.Fatalf("Error adding volume: %v", err)
	}

	t.Cleanup(func() {
		require.NoError(t, knuu.BatchDestroy(executor.Instance, validator, full))
	})

	// Test logic

	err = validator.Start()
	if err != nil {
		t.Fatalf("Error starting instance: %v", err)
	}
	err = validator.WaitInstanceIsRunning()
	if err != nil {
		t.Fatalf("Error waiting for instance to be running: %v", err)
	}
	persistentPeers, err := utils.GetPersistentPeers(executor, []*knuu.Instance{validator})
	if err != nil {
		t.Fatalf("Error getting persistent peers: %v", err)
	}
	err = full.SetArgs("start", "--home=/home/celestia", "--rpc.laddr=tcp://0.0.0.0:26657", "--minimum-gas-prices=0.002utia", "--p2p.persistent_peers", persistentPeers)
	if err != nil {
		t.Fatalf("Error setting args: %v", err)
	}
	t.Log("Starting full node")
	err = full.Start()
	if err != nil {
		t.Fatalf("Error starting instance: %v", err)
	}
	err = full.WaitInstanceIsRunning()
	if err != nil {
		t.Fatalf("Error waiting for instance to be running: %v", err)
	}

	t.Log("Waiting for full node to reach height 1")
	// Wait until full node reaches block height 1 or more
	err = utils.WaitForHeight(executor, full, 5)
	if err != nil {
		t.Fatalf("Error waiting for height: %v", err)
	}

	t.Log("Stopping full node")
	// Stop full node
	err = full.Stop()
	if err != nil {
		t.Fatalf("Error stopping instance: %v", err)
	}

	// Wait until full node is stopped
	err = full.WaitInstanceIsStopped()
	if err != nil {
		t.Fatalf("Error waiting for instance to be stopped: %v", err)
	}

	t.Log("Waiting for 5 seconds")
	// Sleep for 5 seconds
	time.Sleep(5 * time.Second)

	t.Log("Starting full node again")
	// Start full node again
	err = full.Start()
	if err != nil {
		t.Fatalf("Error starting instance: %v", err)
	}

	// Wait until full node is running
	err = full.WaitInstanceIsRunning()
	if err != nil {
		t.Fatalf("Error waiting for instance to be running: %v", err)
	}

	// Get full node height
	height, err := utils.GetHeight(executor, full)

	t.Log("Check if full node height is above 5")
	// Fail if full node height is below 5
	if err != nil {
		t.Fatalf("Error getting height: %v", err)
	}
	if height < 5 {
		t.Fatalf("Full node height is %d, expected at least 5", height)
	}

	t.Log("Waiting for full node to increase height")
	// Wait for full node to reach height + 1
	err = utils.WaitForHeight(executor, full, height+1)
	if err != nil {
		t.Fatalf("Error waiting for height: %v", err)
	}

}
