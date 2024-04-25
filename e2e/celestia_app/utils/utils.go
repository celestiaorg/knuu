package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/knuu/pkg/knuu"
)

type JSONRPCError struct {
	Code    int
	Message string
	Data    string
}

func (e *JSONRPCError) Error() string {
	return fmt.Sprintf("JSONRPC Error - Code: %d, Message: %s, Data: %s", e.Code, e.Message, e.Data)
}

// getStatus returns the status of the node
func getStatus(executor *knuu.Executor, app *knuu.Instance) (string, error) {
	nodeIP, err := app.GetIP()
	if err != nil {
		return "", fmt.Errorf("error getting node ip: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	status, err := executor.ExecuteCommandWithContext(ctx, "wget", "-q", "-O", "-", fmt.Sprintf("%s:26657/status", nodeIP))
	if err != nil {
		return "", fmt.Errorf("error executing command: %w", err)
	}
	return status, nil
}

func NodeIdFromNode(executor *knuu.Executor, node *knuu.Instance) (string, error) {
	status, err := getStatus(executor, node)
	if err != nil {
		return "", fmt.Errorf("error getting status: %v", err)
	}

	id, err := nodeIdFromStatus(status)
	if err != nil {
		return "", fmt.Errorf("error getting node id: %v", err)
	}
	return id, nil
}

func GetHeight(executor *knuu.Executor, app *knuu.Instance) (int64, error) {
	status, err := getStatus(executor, app)
	if err != nil {
		return 0, fmt.Errorf("error getting status: %v", err)
	}
	blockHeight, err := latestBlockHeightFromStatus(status)
	if err != nil {
		return 0, fmt.Errorf("error getting block height: %w", err)
	}
	return blockHeight, nil
}

func WaitForHeight(executor *knuu.Executor, app *knuu.Instance, height int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	return WaitForHeightWithContext(ctx, executor, app, height)
}

func WaitForHeightWithContext(ctx context.Context, executor *knuu.Executor, app *knuu.Instance, height int64) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				return fmt.Errorf("operation canceled: %v", ctx.Err())
			}
			return nil
		case <-ticker.C:
			status, err := getStatus(executor, app)
			if err != nil {
				return fmt.Errorf("error getting status: %v", err)
			}

			blockHeight, err := latestBlockHeightFromStatus(status)
			if err != nil {
				if _, ok := err.(*JSONRPCError); ok {
					// Retry if it's a temporary API error
					continue
				}
				return fmt.Errorf("error getting block height: %w", err)
			}

			if blockHeight >= height {
				return nil
			}
		}
	}
}

func ChainId(executor *knuu.Executor, app *knuu.Instance) (string, error) {
	status, err := getStatus(executor, app)
	if err != nil {
		return "", fmt.Errorf("error getting status: %v", err)
	}
	chainId, err := chainIdFromStatus(status)
	if err != nil {
		return "", fmt.Errorf("error getting chain id: %w", err)
	}
	return chainId, nil
}

func GenesisHash(executor *knuu.Executor, app *knuu.Instance) (string, error) {
	appIP, err := app.GetIP()
	if err != nil {
		return "", fmt.Errorf("error getting app ip: %w", err)
	}
	block, err := executor.ExecuteCommand("wget", "-q", "-O", "-", fmt.Sprintf("%s:26657/block?height=1", appIP))
	if err != nil {
		return "", fmt.Errorf("error getting block: %v", err)
	}
	genesisHash, err := hashFromBlock(block)
	if err != nil {
		return "", fmt.Errorf("error getting hash from block: %v", err)
	}
	return genesisHash, nil
}

func GetPersistentPeers(executor *knuu.Executor, apps []*knuu.Instance) (string, error) {
	var persistentPeers string
	for _, app := range apps {
		validatorIP, err := app.GetIP()
		if err != nil {
			return "", fmt.Errorf("error getting validator IP: %v", err)
		}
		id, err := NodeIdFromNode(executor, app)
		if err != nil {
			return "", fmt.Errorf("error getting node id: %v", err)
		}
		persistentPeers += id + "@" + validatorIP + ":26656" + ","
	}
	return persistentPeers[:len(persistentPeers)-1], nil
}
