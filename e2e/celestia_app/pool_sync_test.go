package celestia_app

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/celestiaorg/knuu/e2e/celestia_app/utils"
	"github.com/celestiaorg/knuu/pkg/knuu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolSync(t *testing.T) {
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

	// new InstancePool struct
	fullNodes := &knuu.InstancePool{}

	t.Cleanup(func() {
		// Cleanup
		if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
			t.Log("Skipping cleanup")
			return
		}

		err = executor.Destroy()
		if err != nil {
			t.Fatalf("Error destroying executor: %v", err)
		}

		err = validator.Destroy()
		if err != nil {
			t.Fatalf("Error destroying validator: %v", err)
		}
		err = full.Destroy()
		if err != nil {
			t.Fatalf("Error destroying full: %v", err)
		}
		err = fullNodes.Destroy()
		if err != nil {
			t.Fatalf("Error destroying full nodes: %v", err)
		}
	})

	// Test logic

	err = validator.Start()
	if err != nil {
		t.Fatalf("Error starting validator: %v", err)
	}
	err = validator.WaitInstanceIsRunning()
	if err != nil {
		t.Fatalf("Error waiting for validator to be running: %v", err)
	}

	persistentPeers, err := utils.GetPersistentPeers(executor, []*knuu.Instance{validator})
	if err != nil {
		t.Fatalf("Error getting persistent peers: %v", err)
	}
	err = full.SetArgs("start", "--home=/home/celestia", "--rpc.laddr=tcp://0.0.0.0:26657", "--minimum-gas-prices=0.002utia", "--p2p.persistent_peers", persistentPeers)
	if err != nil {
		t.Fatalf("Error setting args: %v", err)
	}
	fullNodes, err = full.CreatePool(5)
	if err != nil {
		t.Fatalf("Error creating pool: %v", err)
	}

	err = fullNodes.Start()
	if err != nil {
		t.Fatalf("Error starting full nodes: %v", err)
	}
	err = fullNodes.WaitInstancePoolIsRunning()
	if err != nil {
		t.Fatalf("Error waiting for full nodes to be running: %v", err)
	}

	// Wait until validator reaches block height 3 or more
	t.Log("Waiting for validator to reach block height 3")
	err = utils.WaitForHeight(executor, validator, 3)
	if err != nil {
		t.Fatalf("Error waiting for validator to reach block height 3: %v", err)
	}

	// Wait until full node reaches block height 3 or more but error out if it takes too long
	t.Log("Waiting for full nodes to reach block height 3")
	for _, full := range fullNodes.Instances() {
		err = utils.WaitForHeight(executor, full, 3)
		if err != nil {
			t.Fatalf("Error waiting for full node to reach block height 3: %v", err)
		}
	}
}

// TestPoolSync_WithTrafficShape tests if a restricted full node takes longer to sync than a full node
// This is just a sample code to demonstrate how to use the bandwidth shaping feature
// Feel free to modify it to suit your needs
func TestPoolSync_WithTrafficShape(t *testing.T) {
	t.Parallel()
	// Setup

	const targetHeight = 25

	executor, err := knuu.NewExecutor()
	require.NoError(t, err, "Error creating executor")

	validator, err := Instances["validator"].Clone()
	require.NoError(t, err, "Error cloning validator instance")

	full, err := Instances["full"].CloneWithName("full")
	require.NoError(t, err, "Error cloning full node instance")

	fullRestricted, err := Instances["full"].CloneWithName("full-restricted")
	require.NoError(t, err, "Error cloning restricted full node instance")

	t.Cleanup(func() {
		// Cleanup
		if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
			t.Log("Skipping cleanup")
			return
		}

		require.NoError(t, executor.Destroy(), "Error destroying executor")

		// These checks are added to avoid errors when the test fails
		// before the instances are started
		if validator.IsInState(knuu.Started) {
			require.NoError(t, validator.Destroy(), "Error destroying validator")
		}

		if full.IsInState(knuu.Started) {
			require.NoError(t, full.Destroy(), "Error destroying full")
		}

		if fullRestricted.IsInState(knuu.Started) {
			require.NoError(t, fullRestricted.Destroy(), "Error destroying restricted full")
		}
	})

	// Test logic
	require.NoError(t, validator.Start(), "Error starting validator")
	err = utils.WaitForHeight(executor, validator, 1)
	require.NoError(t, err, "Error waiting for validator to reach block height 1")

	persistentPeers, err := utils.GetPersistentPeers(executor, []*knuu.Instance{validator})
	require.NoError(t, err, "Error getting persistent peers")

	err = full.SetArgs("start", "--home=/home/celestia", "--rpc.laddr=tcp://0.0.0.0:26657", "--minimum-gas-prices=0.002utia", "--p2p.persistent_peers", persistentPeers)
	require.NoError(t, err, "Error setting args for full node")

	err = fullRestricted.SetArgs("start", "--home=/home/celestia", "--rpc.laddr=tcp://0.0.0.0:26657", "--minimum-gas-prices=0.002utia", "--p2p.persistent_peers", persistentPeers)
	require.NoError(t, err, "Error setting args for restricted full node")

	// Wait until validator reaches the target block height
	t.Logf("Waiting for validator to reach block height %d", targetHeight)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	err = utils.WaitForHeightWithContext(ctx, executor, validator, targetHeight)
	require.NoError(t, err, "Error waiting for validator to reach block height %d", targetHeight)

	noRestrictionElapsed := time.Duration(0)
	{
		require.NoError(t, full.Start(), "Error starting full node")

		startTime := time.Now().UnixNano()

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
		defer cancel()
		err = utils.WaitForHeightWithContext(ctx, executor, full, targetHeight)
		require.NoError(t, err, "Error waiting for full node to reach block height %d", targetHeight)

		endTime := time.Now().UnixNano()
		noRestrictionElapsed = time.Duration(endTime - startTime)
		t.Logf("Elapsed time for full node without traffic shaping: %f seconds", noRestrictionElapsed.Seconds())
	}

	restrictedElapsed := time.Duration(0)
	{
		require.NoError(t, fullRestricted.EnableBitTwister(), "Error enabling BitTwister")
		require.NoError(t, fullRestricted.Start(), "Error starting restricted full node")
		forwardBitTwisterPort(t, fullRestricted)

		startTime := time.Now().UnixNano()

		// Set bandwidth limit for Restricted full node and then sync it with the validator
		err = fullRestricted.SetBandwidthLimit(10 * 100) // 1kbps
		require.NoError(t, err, "Error setting bandwidth limit for fullRestricted")

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
		defer cancel()
		err = utils.WaitForHeightWithContext(ctx, executor, fullRestricted, targetHeight)
		require.NoError(t, err, "Error waiting for restricted full node to reach block height %d", targetHeight)

		endTime := time.Now().UnixNano()
		restrictedElapsed = time.Duration(endTime - startTime)
		t.Logf("Elapsed time for restricted full node with traffic shaping: %f seconds", restrictedElapsed.Seconds())
	}

	// Check if restricted full node took longer than full node
	assert.Greater(t, restrictedElapsed, noRestrictionElapsed, "full node took longer than restricted full node to sync")
}
