package main

import (
	"encoding/json"
	"fmt"
	"github.com/celestiaorg/knuu/pkg/knuu"
	"io"
	"math/rand"
	"regexp"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

type keyGenerator struct {
	random *rand.Rand
}

func newKeyGenerator(seed int64) *keyGenerator {
	return &keyGenerator{
		random: rand.New(rand.NewSource(seed)), //nolint:gosec
	}
}

func (g *keyGenerator) Generate(keyType string) crypto.PrivKey {
	seed := make([]byte, ed25519.SeedSize)

	_, err := io.ReadFull(g.random, seed)
	if err != nil {
		panic(err) // this shouldn't happen
	}
	switch keyType {
	case "secp256k1":
		return secp256k1.GenPrivKeySecp256k1(seed)
	case "", "ed25519":
		return ed25519.GenPrivKeyFromSecret(seed)
	default:
		panic("KeyType not supported") // should not make it this far
	}
}

// getStatus returns the status of the node
func getStatus(executor *knuu.Executor, app *knuu.Instance) (string, error) {
	nodeIP, err := app.GetIP()
	if err != nil {
		return "", fmt.Errorf("error getting node ip: %w", err)
	}
	status, err := executor.ExecuteCommand("wget", "-q", "-O", "-", fmt.Sprintf("%s:26657/status", nodeIP))
	if err != nil {
		return "", fmt.Errorf("error executing command: %w", err)
	}
	return status, nil
}

func chainIdFromStatus(status string) (string, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(status), &result)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling status: %w", err)
	}
	resultData, ok := result["result"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("error getting result from status")
	}
	nodeInfo, ok := resultData["node_info"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("error getting node info from status")
	}
	chainId, ok := nodeInfo["network"].(string)
	if !ok {
		return "", fmt.Errorf("error getting network from node info")
	}
	return chainId, nil
}

func hashFromBlock(block string) (string, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(block), &result)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling block: %w", err)
	}
	resultData, ok := result["result"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("error getting result from block")
	}
	blockId, ok := resultData["block_id"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("error getting block id from block")
	}
	blockHash, ok := blockId["hash"].(string)
	if !ok {
		return "", fmt.Errorf("error getting hash from block id")
	}
	return blockHash, nil
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

func authTokenFromAuth(auth string) (string, error) {
	// Use regex to match the JWT token
	re := regexp.MustCompile(`[A-Za-z0-9\-_=]+\.[A-Za-z0-9\-_=]+\.?[A-Za-z0-9\-_=]*`)
	match := re.FindString(auth)

	return fmt.Sprintf(match), nil
}

func iDFromP2PInfo(p2pInfo string) (string, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(p2pInfo), &result)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling status: %w", err)
	}
	resultData := result["result"].(map[string]interface{})
	id := resultData["ID"].(string)
	return id, nil
}
