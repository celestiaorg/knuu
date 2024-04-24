package celestia_app

import (
	"context"
	"os"
	"testing"

	"github.com/celestiaorg/knuu/pkg/knuu"

	app_utils "github.com/celestiaorg/knuu-example/celestia_app/utils"
	"github.com/celestiaorg/knuu-example/celestia_node/utils"
)

func TestBasic(t *testing.T) {
	t.Parallel()
	// Setup

	executor, err := knuu.NewExecutor()
	if err != nil {
		t.Fatalf("Error creating executor: %v", err)
	}

	t.Log("Starting consensus")
	consensus, err := app_utils.CreateAndStartConsensus(executor)
	if err != nil {
		t.Fatalf("Error creating and starting consensus: %v", err)
	}

	t.Log("Starting bridge")
	bridge, err := utils.CreateAndStartBridge(executor, "bridge", consensus)

	t.Log("Starting full node")
	full, err := utils.CreateAndStartNode(executor, "full", "full", consensus, bridge)

	t.Log("Starting light node")
	light, err := utils.CreateAndStartNode(executor, "light", "light", consensus, full)

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
		err = consensus.Destroy()
		if err != nil {
			t.Fatalf("Error destroying executor: %v", err)
		}
		err = bridge.Destroy()
		if err != nil {
			t.Fatalf("Error destroying instance: %v", err)
		}
		err = full.Destroy()
		if err != nil {
			t.Fatalf("Error destroying instance: %v", err)
		}
		err = light.Destroy()
		if err != nil {
			t.Fatalf("Error destroying instance: %v", err)
		}
	})

	// Test logic

	rpcClientBridge, err := utils.GetRPCClient(full)
	if err != nil {
		t.Fatalf("Error getting rpc client: %v", err)
	}
	rpcClientFull, err := utils.GetRPCClient(full)
	if err != nil {
		t.Fatalf("Error getting rpc client: %v", err)
	}
	rpcClientLight, err := utils.GetRPCClient(full)
	if err != nil {
		t.Fatalf("Error getting rpc client: %v", err)
	}

	infoBridge, err := rpcClientBridge.Node.Info(context.TODO())
	if err != nil {
		t.Fatalf("Error getting node info: %v", err)
	}
	t.Logf("Bridge info: %s, %t, %s", infoBridge.Type.String(), infoBridge.Type.IsValid(), infoBridge.APIVersion)
	infoFull, err := rpcClientFull.Node.Info(context.TODO())
	if err != nil {
		t.Fatalf("Error getting node info: %v", err)
	}
	t.Logf("Full info: %s, %t, %s", infoFull.Type.String(), infoFull.Type.IsValid(), infoFull.APIVersion)
	infoLight, err := rpcClientLight.Node.Info(context.TODO())
	if err != nil {
		t.Fatalf("Error getting node info: %v", err)
	}
	t.Logf("Light info: %s, %t, %s", infoLight.Type.String(), infoLight.Type.IsValid(), infoLight.APIVersion)
}
