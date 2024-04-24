package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/celestiaorg/knuu/pkg/knuu"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
)

const (
	dockerPackageTxsim = "ghcr.io/celestiaorg/txsim"
)

type Txsim struct {
	Name       string
	Version    string
	SignerKey  crypto.PrivKey
	NetworkKey crypto.PrivKey
	AccountKey crypto.PrivKey
	Instance   *knuu.Instance

	RPCEndpoints  []string
	GRPCEndpoints []string
	PollTime      time.Duration
	BlobSizes     []int
	Blob          int
	BlobAmounts   int
	Seed          int
	Send          int
}

func NewTxsimNode(
	name, version string,
	signerKey, networkKey, accountKey crypto.PrivKey,
	mnemomic string,
	rpcEndpoints, grpcEndpoints []string,
	pollTime time.Duration,
	blobSizes []int,
	blob, blobAmounts, seed, send int,
) (*Txsim, error) {

	instance, err := knuu.NewInstance(name)
	if err != nil {
		return nil, err
	}
	err = instance.SetImage(fmt.Sprintf("%s:%s", dockerPackageTxsim, version))
	if err != nil {
		return nil, err
	}
	err = instance.SetMemory("200Mi", "200Mi")
	if err != nil {
		return nil, err
	}
	err = instance.SetCPU("300m")
	if err != nil {
		return nil, err
	}
	err = instance.AddVolumeWithOwner(remoteRootDir, persistentVolumeSize, 10001)
	if err != nil {
		return nil, err
	}
	err = instance.SetCommand("/bin/txsim")
	if err != nil {
		return nil, err
	}
	if len(blobSizes) != 2 {
		return nil, fmt.Errorf("blob sizes must be a slice of two integers")
	}
	blobSizesString := fmt.Sprintf("%d-%d", blobSizes[0], blobSizes[1])
	err = instance.SetArgs(
		"--key-mnemonic",
		fmt.Sprintf("%s", mnemomic),
		"--rpc-endpoints",
		strings.Join(rpcEndpoints, ","),
		"--grpc-endpoints",
		strings.Join(grpcEndpoints, ","),
		"--poll-time",
		pollTime.String(),
		"--blob-sizes",
		blobSizesString,
		"--blob",
		fmt.Sprintf("%d", blob),
		"--blob-amounts",
		fmt.Sprintf("%d", blobAmounts),
		"--seed",
		fmt.Sprintf("%d", seed),
		"--send",
		fmt.Sprintf("%d", send),
	)
	if err != nil {
		return nil, err
	}
	_, err = instance.ExecuteCommand(fmt.Sprintf("mkdir -p %s/config", remoteRootDir))
	if err != nil {
		return nil, err
	}
	_, err = instance.ExecuteCommand(fmt.Sprintf("mkdir -p %s/data", remoteRootDir))
	if err != nil {
		return nil, err
	}
	err = instance.Commit()
	if err != nil {
		return nil, err
	}

	return &Txsim{
		Name:       name,
		Instance:   instance,
		Version:    version,
		SignerKey:  signerKey,
		NetworkKey: networkKey,
		AccountKey: accountKey,

		RPCEndpoints:  rpcEndpoints,
		GRPCEndpoints: grpcEndpoints,
		PollTime:      pollTime,
		BlobSizes:     blobSizes,
		Blob:          blob,
		BlobAmounts:   blobAmounts,
		Seed:          seed,
	}, nil
}

func (ts *Txsim) Init() error {
	// Initialize file directories
	rootDir := os.TempDir()
	nodeDir := filepath.Join(rootDir, ts.Name)
	for _, dir := range []string{
		filepath.Join(nodeDir, "config"),
		filepath.Join(nodeDir, "data"),
	} {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return fmt.Errorf("error creating directory %s: %w", dir, err)
		}
	}

	// Store the node key for the p2p handshake
	nodeKeyFilePath := filepath.Join(nodeDir, "config", "node_key.json")
	err := (&p2p.NodeKey{PrivKey: ts.NetworkKey}).SaveAs(nodeKeyFilePath)
	if err != nil {
		return err
	}

	err = os.Chmod(nodeKeyFilePath, 0o777)
	if err != nil {
		return fmt.Errorf("chmod node key: %w", err)
	}

	// Store the validator signer key for consensus
	pvKeyPath := filepath.Join(nodeDir, "config", "priv_validator_key.json")
	pvStatePath := filepath.Join(nodeDir, "data", "priv_validator_state.json")
	(privval.NewFilePV(ts.SignerKey, pvKeyPath, pvStatePath)).Save()

	err = ts.Instance.AddFile(pvKeyPath, filepath.Join(remoteRootDir, "config", "priv_validator_key.json"), "10001:10001")
	if err != nil {
		return fmt.Errorf("adding priv_validator_key file: %w", err)
	}

	err = ts.Instance.AddFile(pvStatePath, filepath.Join(remoteRootDir, "data", "priv_validator_state.json"), "10001:10001")
	if err != nil {
		return fmt.Errorf("adding priv_validator_state file: %w", err)
	}

	err = ts.Instance.AddFile(nodeKeyFilePath, filepath.Join(remoteRootDir, "config", "node_key.json"), "10001:10001")
	if err != nil {
		return fmt.Errorf("adding node_key file: %w", err)
	}

	return nil
}

func (ts *Txsim) Start() error {
	if err := ts.Instance.Start(); err != nil {
		return err
	}

	if err := ts.Instance.WaitInstanceIsRunning(); err != nil {
		return err
	}

	return nil
}
