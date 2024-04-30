package utils

import (
	"fmt"

	"github.com/celestiaorg/knuu/pkg/knuu"
)

const consImg = "ghcr.io/celestiaorg/celestia-app:v1.7.0"

func CreateAndStartConsensus(executor *knuu.Executor) (*knuu.Instance, error) {
	consensus, err := knuu.NewInstance("consensus")
	if err != nil {
		return nil, fmt.Errorf("error creating instance: %w", err)
	}
	err = consensus.SetImage(consImg)
	if err != nil {
		return nil, fmt.Errorf("error setting image: %w", err)
	}
	err = consensus.AddPortTCP(26656)
	if err != nil {
		return nil, fmt.Errorf("error adding port: %w", err)
	}
	err = consensus.AddPortTCP(26657)
	if err != nil {
		return nil, fmt.Errorf("error adding port: %w", err)
	}
	err = consensus.AddPortTCP(9090)
	if err != nil {
		return nil, fmt.Errorf("error adding port: %w", err)
	}
	err = consensus.AddFile("resources/genesis.sh", "/opt/genesis.sh", "0:0")
	if err != nil {
		return nil, fmt.Errorf("error adding file: %w", err)
	}
	_, err = consensus.ExecuteCommand("/bin/sh", "/opt/genesis.sh")
	if err != nil {
		return nil, fmt.Errorf("error executing command: %w", err)
	}
	err = consensus.SetArgs("start", "--rpc.laddr=tcp://0.0.0.0:26657", "--api.enable", "--grpc.enable")
	if err != nil {
		return nil, fmt.Errorf("error setting args: %w", err)
	}
	err = consensus.Commit()
	if err != nil {
		return nil, fmt.Errorf("error committing instance: %w", err)
	}

	err = consensus.Start()
	if err != nil {
		return nil, err
	}
	err = consensus.WaitInstanceIsRunning()
	if err != nil {
		return nil, err
	}

	// Wait until validator reaches block height 1 or more
	err = WaitForHeight(executor, consensus, 1)
	if err != nil {
		return nil, err
	}

	return consensus, nil
}
