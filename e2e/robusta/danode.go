package main

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/celestiaorg/knuu/pkg/knuu"
)

const (
	daNodeDockerSrcURL         = "ghcr.io/celestiaorg/celestia-node"
	daNodeRemoteRootDir        = "/home/celestia/.celestia-node"
	daNodePersistentVolumeSize = "100Gi"
)

// daNodeType enum (bridge, full, light)
type daNodeType string

const (
	bridge daNodeType = "bridge"
	full              = "full"
	light             = "light"
)

// daNodeType to string
func (t daNodeType) String() string {
	return string(t)
}

type DaNode struct {
	Name         string
	Version      string
	DaNodeType   daNodeType
	IP           string
	CoreIP       string
	trustedPeers []string
	Instance     *knuu.Instance
}

func NewDaNode(
	name, version string,
	executor *knuu.Executor,
	nodeType daNodeType,
	consensusNode *Node,
	trustedPeers []*DaNode,
) (*DaNode, error) {

	consensusNodeRunning, err := consensusNode.Instance.IsRunning()
	if err != nil {
		return nil, err
	}
	if !consensusNodeRunning {
		return nil, fmt.Errorf("consensus node %s is not running", consensusNode.Name)
	}
	for _, trustedPeer := range trustedPeers {
		trustedPeerRunning, err := trustedPeer.Instance.IsRunning()
		if err != nil {
			return nil, err
		}
		if !trustedPeerRunning {
			return nil, fmt.Errorf("trusted peer %s is not running", trustedPeer.Name)
		}
	}

	chainId, err := ChainId(executor, consensusNode.Instance)
	if err != nil {
		return nil, err
	}
	genesisHash, err := GenesisHash(executor, consensusNode.Instance)
	if err != nil {
		return nil, err
	}

	instance, err := knuu.NewInstance(name)
	if err != nil {
		return nil, err
	}
	err = instance.SetImage(fmt.Sprintf("%s:%s", daNodeDockerSrcURL, version))
	if err != nil {
		return nil, err
	}
	err = instance.AddPortTCP(2121)
	if err != nil {
		return nil, err
	}
	err = instance.AddPortTCP(26658)
	if err != nil {
		return nil, err
	}
	err = instance.SetMemory("400Mi", "400Mi")
	if err != nil {
		return nil, err
	}
	err = instance.SetCPU("300m")
	if err != nil {
		return nil, err
	}
	err = instance.AddVolumeWithOwner(daNodeRemoteRootDir, daNodePersistentVolumeSize, 10001)
	if err != nil {
		return nil, err
	}
	err = instance.SetEnvironmentVariable("CELESTIA_CUSTOM", fmt.Sprintf("%s:%s", chainId, genesisHash))
	if err != nil {
		return nil, err
	}
	_, err = instance.ExecuteCommand("celestia", nodeType.String(), "init", "--node.store", daNodeRemoteRootDir)
	if err != nil {
		return nil, err
	}
	err = instance.SetUser("10001")
	if err != nil {
		return nil, err
	}

	if observabilityEnabled {
		if err := instance.SetOtelEndpoint(4318); err != nil {
			return nil, err
		}
		if err := instance.SetPrometheusEndpoint(8890, fmt.Sprintf("knuu-%s", knuu.Scope()), "1m"); err != nil {
			return nil, err
		}
		if err := instance.SetJaegerEndpoint(14250, 6831, 14268); err != nil {
			return nil, err
		}
		if err := instance.SetOtlpExporter(grafanaEndpoint, grafanaUsername, grafanaToken); err != nil {
			return nil, err
		}
		if err := instance.SetJaegerExporter("jaeger-collector.jaeger-cluster.svc.cluster.local:14250"); err != nil {
			return nil, err
		}
	}

	consensusIP, err := consensusNode.Instance.GetIP()
	if err != nil {
		return nil, err
	}

	commonArgs := []string{
		"celestia",
		nodeType.String(),
		"start",
		"--node.store", daNodeRemoteRootDir,
		"--core.ip", consensusIP,
	}

	observabilityArgs := []string{
		"--metrics",
		"--metrics.tls=false",
		"--p2p.metrics",
		"--tracing",
		"--tracing.tls=false",
	}

	if observabilityEnabled {
		commonArgs = append(commonArgs, observabilityArgs...)
	}

	if len(trustedPeers) == 0 {
		err = instance.SetCommand(commonArgs...)
		if err != nil {
			return nil, err
		}
	} else {
		var trustedPeersArray []string
		for _, peer := range trustedPeers {
			peerIP, err := peer.Instance.GetIP()
			if err != nil {
				return nil, err
			}
			adminAuthToken, err := peer.Instance.ExecuteCommand("celestia", peer.DaNodeType.String(), "auth", "admin", "--node.store", daNodeRemoteRootDir)
			if err != nil {
				return nil, err
			}
			adminAuthTokenString, err := authTokenFromAuth(adminAuthToken)
			if err != nil {
				return nil, err
			}
			p2pInfo, err := peer.Instance.ExecuteCommand("celestia", "rpc", "p2p", "Info", "--auth", adminAuthTokenString)
			if err != nil {
				return nil, err
			}
			peerID, err := iDFromP2PInfo(p2pInfo)
			if err != nil {
				return nil, err
			}

			trustedPeersArray = append(trustedPeersArray, fmt.Sprintf("/ip4/%s/tcp/2121/p2p/%s", peerIP, peerID))
		}

		trustedPeersString := strings.Join(trustedPeersArray, ",")
		commonArgs = append(commonArgs, "--headers.trusted-peers", trustedPeersString)

		err = instance.SetCommand(commonArgs...)
		if err != nil {
			return nil, err
		}
	}

	return &DaNode{
		Name:       name,
		Instance:   instance,
		Version:    version,
		DaNodeType: nodeType,
		CoreIP:     consensusIP,
	}, nil
}

func (n *DaNode) Init() error {
	log.Info().Str("name", n.Name).Msg("Initializing node")

	// FIXME: if you commit before adding files, we can cache instances and save build time
	err := n.Instance.Commit()
	if err != nil {
		return err
	}

	log.Info().Str("name", n.Name).Msg("Initialized node")

	return nil
}

func (n *DaNode) Start() error {
	log.Info().Str("name", n.Name).Msg("Starting node")

	if err := n.Instance.Start(); err != nil {
		return err
	}

	if err := n.Instance.WaitInstanceIsRunning(); err != nil {
		return err
	}

	log.Info().Str("name", n.Name).Msg("Started node")
	return nil
}
