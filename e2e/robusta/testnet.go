package main

import (
	"fmt"
	"os"
	"time"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/knuu/pkg/knuu"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/rs/zerolog/log"
)

type Testnet struct {
	seed            int64
	nodes           []*Node
	daNodes         []*DaNode
	genesisAccounts []*GenesisAccount
	keygen          *keyGenerator
	Txsim           *Txsim
	executor        *knuu.Executor
}

func New(name string, seed int64) (*Testnet, error) {
	identifier := fmt.Sprintf("%s_%s", name, time.Now().Format("20060102_150405"))
	if err := knuu.InitializeWithIdentifier(identifier); err != nil {
		return nil, err
	}

	executor, err := knuu.NewExecutor()
	if err != nil {
		return nil, err
	}

	return &Testnet{
		seed:            seed,
		nodes:           make([]*Node, 0),
		daNodes:         make([]*DaNode, 0),
		genesisAccounts: make([]*GenesisAccount, 0),
		keygen:          newKeyGenerator(seed),
		Txsim:           nil,
		executor:        executor,
	}, nil
}

func (t *Testnet) CreateGenesisNode(version string, selfDelegation int64) (*Node, error) {
	signerKey := t.keygen.Generate(ed25519Type)
	networkKey := t.keygen.Generate(ed25519Type)
	accountKey := t.keygen.Generate(secp256k1Type)
	nodeName := fmt.Sprintf("val%d", len(t.nodes))
	log.Info().Str("name", nodeName).Msg("Creating node")
	node, err := NewNode(nodeName, version, 0, selfDelegation, nil, signerKey, networkKey, accountKey)
	if err != nil {
		return nil, err
	}
	t.nodes = append(t.nodes, node)
	log.Info().Str("name", nodeName).Msg("Created node")
	return node, nil
}

func (t *Testnet) CreateGenesisNodes(num int, version string, selfDelegation int64) ([]*Node, error) {
	nodes := make([]*Node, num)
	for i := -0; i < num; i++ {
		node, err := t.CreateGenesisNode(version, selfDelegation)
		if err != nil {
			return nil, err
		}
		nodes[i] = node
	}
	return nodes, nil
}

func (t *Testnet) CreateNode(version string, startHeight int64) (*Node, error) {
	signerKey := t.keygen.Generate(ed25519Type)
	networkKey := t.keygen.Generate(ed25519Type)
	accountKey := t.keygen.Generate(secp256k1Type)
	nodeName := fmt.Sprintf("full%d", len(t.nodes))
	log.Info().Str("name", nodeName).Msg("Creating node")
	node, err := NewNode(nodeName, version, startHeight, 0, nil, signerKey, networkKey, accountKey)
	if err != nil {
		return nil, err
	}
	t.nodes = append(t.nodes, node)
	log.Info().Str("name", nodeName).Msg("Created node")
	return node, nil
}

func (t *Testnet) CreateNodes(num int, version string, startHeight int64) ([]*Node, error) {
	nodes := make([]*Node, num)
	for i := 0; i < num; i++ {
		node, err := t.CreateNode(version, startHeight)
		if err != nil {
			return nil, err
		}
		nodes[i] = node
	}
	return nodes, nil
}

func (t *Testnet) CreateDaNode(nodeType daNodeType, version string, consensusNode *Node, trustedPeers []*DaNode) (*DaNode, error) {
	nodeName := fmt.Sprintf("da-%s%d", nodeType, len(t.daNodes))
	log.Info().Str("name", nodeName).Msg("Creating da node")
	daNode, err := NewDaNode(nodeName, version, t.executor, nodeType, consensusNode, trustedPeers)
	if err != nil {
		return nil, err
	}
	t.daNodes = append(t.daNodes, daNode)
	log.Info().Str("name", nodeName).Msg("Created da node")
	return daNode, nil

}

func (t *Testnet) CreateTxSim(
	mnemomic, version string,
	pollTime time.Duration,
	blobSizes []int,
	blob, blobAmounts, seed, send int,
) error {
	signerKey := t.keygen.Generate(ed25519Type)
	networkKey := t.keygen.Generate(ed25519Type)
	accountKey := t.keygen.Generate(secp256k1Type)

	txSimNode, err := NewTxsimNode("txsim", version, signerKey, networkKey, accountKey, mnemomic, t.ExternalRPCEndpoints(), t.ExternalGRPCEndpoints(), pollTime, blobSizes, blob, blobAmounts, seed, send)
	if err != nil {
		return err
	}
	t.Txsim = txSimNode
	return nil
}

func (t *Testnet) CreateGenesisAccount(name string, tokens int64) (keyring.Keyring, string, error) {
	cdc := encoding.MakeConfig(app.ModuleEncodingRegisters...).Codec
	kr := keyring.NewInMemory(cdc)
	key, mnemomic, err := kr.NewMnemonic(name, keyring.English, "", "", hd.Secp256k1)
	if err != nil {
		return nil, "", err
	}
	pk, err := key.GetPubKey()
	if err != nil {
		return nil, "", err
	}
	t.genesisAccounts = append(t.genesisAccounts, &GenesisAccount{
		PubKey:        pk,
		InitialTokens: tokens,
	})
	return kr, mnemomic, nil
}

func (t *Testnet) Setup() error {
	genesisNodes := make([]*Node, 0)
	for _, node := range t.nodes {
		if node.StartHeight == 0 && node.SelfDelegation > 0 {
			genesisNodes = append(genesisNodes, node)
		}
	}
	genesis, err := MakeGenesis(genesisNodes, t.genesisAccounts)
	if err != nil {
		return err
	}
	for _, node := range t.nodes {

		// nodes are initialized with the addresses of all other
		// nodes in their addressbook
		peers := make([]string, 0, len(t.nodes)-1)
		for _, peer := range t.nodes {
			if peer.Name != node.Name {
				peers = append(peers, peer.AddressP2P(true))
			}
		}

		err = node.Init(genesis, peers)
		if err != nil {
			return err
		}
	}

	for _, daNode := range t.daNodes {
		err = daNode.Init()
		if err != nil {
			return err
		}
	}

	if t.Txsim != nil {
		err := t.Txsim.Init()
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Testnet) ExternalRPCEndpoints() []string {
	rpcEndpoints := make([]string, len(t.nodes))
	for idx, node := range t.nodes {
		rpcEndpoints[idx] = node.ExternalAddressRPC()
	}
	return rpcEndpoints
}

func (t *Testnet) ExternalGRPCEndpoints() []string {
	grpcEndpoints := make([]string, len(t.nodes))
	for idx, node := range t.nodes {
		grpcEndpoints[idx] = node.ExternalAddressGRPC()
	}
	return grpcEndpoints
}

func (t *Testnet) Start() error {
	genesisNodes := make([]*Node, 0)
	for _, node := range t.nodes {
		if node.StartHeight == 0 {
			genesisNodes = append(genesisNodes, node)
		}
	}
	for _, node := range genesisNodes {
		err := node.Start()
		if err != nil {
			return fmt.Errorf("node %s failed to start: %w", node.Name, err)
		}
	}
	for _, node := range t.nodes {
		if node.StartHeight > 0 {
			err := node.Start()
			if err != nil {
				return fmt.Errorf("node %s failed to start: %w", node.Name, err)
			}
		}
	}

	for _, daNode := range t.daNodes {
		err := daNode.Start()
		if err != nil {
			return fmt.Errorf("node %s failed to start: %w", daNode.Name, err)
		}
	}

	if t.Txsim != nil {
		err := t.Txsim.Start()
		if err != nil {
			return fmt.Errorf("failed to start txsim: %w", err)
		}
	}
	return nil
}

func (t *Testnet) Cleanup() {
	if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
		log.Info().Msg("skipping cleanup")
		return
	}
	for _, node := range t.nodes {
		err := node.Instance.Destroy()
		if err != nil {
			log.Err(err).Msg(fmt.Sprintf("node %s failed to cleanup", node.Name))
		}
	}
}

func (t *Testnet) Node(i int) *Node {
	return t.nodes[i]
}
