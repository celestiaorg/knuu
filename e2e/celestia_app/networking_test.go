package celestia_app

import (
	"testing"
	"time"

	"github.com/celestiaorg/knuu/e2e/celestia_app/utils"
	"github.com/celestiaorg/knuu/pkg/knuu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworking_DisableNetwork(t *testing.T) {
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

	t.Cleanup(func() {
		require.NoError(t, knuu.BatchDestroy(executor.Instance, validator, full))
	})

	// Test logic

	require.NoError(t, validator.Start(), "Error starting validator")

	validatorIP, err := validator.GetIP()
	require.NoError(t, err, "Error getting validator IP")

	id, err := utils.NodeIdFromNode(executor, validator)
	require.NoError(t, err, "Error getting node id")

	persistentPeers := id + "@" + validatorIP + ":26656"
	err = full.SetArgs("start", "--home=/home/celestia", "--rpc.laddr=tcp://0.0.0.0:26657", "--minimum-gas-prices=0.002utia", "--p2p.persistent_peers", persistentPeers)
	require.NoError(t, err, "Error setting args")

	t.Log("Starting full nodes")
	require.NoError(t, full.Start(), "Error starting full node")

	// Wait until validator reaches block height 1 or more
	err = utils.WaitForHeight(executor, validator, 1)
	require.NoError(t, err, "Error waiting for height")

	err = utils.WaitForHeight(executor, full, 1)
	require.NoError(t, err, "Error waiting for full node height to be >= 1")

	// Disable networking
	t.Log("Disabling networking")
	require.NoError(t, full.DisableNetwork(), "Error disabling network")

	// Get current block height
	fullNodeHeight, err := utils.GetHeight(executor, full)
	require.NoError(t, err, "Error getting height")

	// Fail if height increases more than 1 for next 1 minute
	t.Log("Waiting for height to not increase for 1 minute")
	timeout := time.After(1 * time.Minute)
	tick := time.Tick(1 * time.Second)

	for {
		select {
		case <-timeout:
			goto afterTimeout
		case <-tick:
			newHeight, err := utils.GetHeight(executor, full)
			require.NoError(t, err, "Error getting height")
			if newHeight > fullNodeHeight+1 {
				t.Fatalf("Height increased from %d to %d", fullNodeHeight, newHeight)
			}
			fullNodeHeight = newHeight
		}
	}

afterTimeout:
	// Enable networking
	t.Log("Enabling networking")
	require.NoError(t, full.EnableNetwork(), "Error enabling network")

	// Wait until validator reaches at least one block higher than what the fullnode has already synced
	err = utils.WaitForHeight(executor, validator, fullNodeHeight+1)
	require.NoError(t, err, "Error waiting for height")

	t.Log("Waiting for 30 seconds to allow the full node to start to sync again...")
	time.Sleep(30 * time.Second)

	height, err := utils.GetHeight(executor, full)
	require.NoError(t, err, "Error getting block height")

	assert.Greater(t, height, fullNodeHeight, "new height should be greater")
}

func TestNetworking_SetPacketLossDynamic(t *testing.T) {
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

	validatorIP, err := validator.GetIP()
	if err != nil {
		t.Fatalf("Error getting validator IP: %v", err)
	}
	id, err := utils.NodeIdFromNode(executor, validator)
	if err != nil {
		t.Fatalf("Error getting node id: %v", err)
	}
	persistentPeers := id + "@" + validatorIP + ":26656"
	err = full.SetArgs("start", "--home=/home/celestia", "--rpc.laddr=tcp://0.0.0.0:26657", "--minimum-gas-prices=0.002utia", "--p2p.persistent_peers", persistentPeers)
	if err != nil {
		t.Fatalf("Error setting args: %v", err)
	}

	t.Log("Starting full nodes")
	require.NoError(t, full.EnableBitTwister(), "Error enabling BitTwister")
	err = full.Start()
	if err != nil {
		t.Fatalf("Error starting instance: %v", err)
	}
	err = full.WaitInstanceIsRunning()
	if err != nil {
		t.Fatalf("Error waiting for instance to be running: %v", err)
	}

	// forward the port to your local
	forwardBitTwisterPort(t, full)

	// Wait until validator reaches block height 1 or more
	err = utils.WaitForHeight(executor, full, 1)
	if err != nil {
		t.Fatalf("Error waiting for height: %v", err)
	}

	// get the current height of the full node
	oldHeight, err := utils.GetHeight(executor, full)
	if err != nil {
		t.Fatalf("Error getting height: %v", err)
	}

	// set the package loss to 100%
	require.NoError(t, full.SetPacketLoss(100), "Error setting packet loss to 100")

	// Fail if height increases more than 1 for next 1 minute
	t.Log("Waiting for 1 minute...")
	time.Sleep(1 * time.Minute)
	t.Log("Done waiting")

	// disable the package loss
	require.NoError(t, full.SetPacketLoss(0), "Error setting packet loss")

	// now it should keep the old height, but start getting updated
	// so tolerance of 5 might be enough depending on the block time
	newHeight, err := utils.GetHeight(executor, full)
	if err != nil {
		t.Fatalf("Error getting height: %v", err)
	}
	assert.LessOrEqual(t, newHeight, oldHeight+5)

	// Check if block height is increasing, timeout after some time
	t.Log("Waiting for height to increase")
	height, err := utils.GetHeight(executor, validator)
	if err != nil {
		t.Fatalf("Error getting block height: %v", err)
	}
	err = utils.WaitForHeight(executor, validator, height+1)
	if err != nil {
		t.Fatalf("Error waiting for height: %v", err)
	}
}
