package main

import (
	"github.com/rs/zerolog/log"
	"os"
)

const (
	appVersion   = "v1.0.0-rc15"
	nodeVersion  = "v0.11.0-rc13"
	txSimVersion = "a9e2acd"
	seed         = 42 // the meaning of life

	observabilityEnabled = false
)

var (
	grafanaEndpoint = ""
	grafanaUsername = ""
	grafanaToken    = ""
)

func main() {

	name := "test"

	testnet, err := New(name, seed)
	if err != nil {
		panic(err)
	}

	if observabilityEnabled {
		grafanaEndpoint = os.Getenv("GRAFANA_ENDPOINT")
		if grafanaEndpoint == "" {
			log.Fatal().Msg("GRAFANA_ENDPOINT env var must be set")
		}
		grafanaUsername = os.Getenv("GRAFANA_USERNAME")
		if grafanaUsername == "" {
			log.Fatal().Msg("GRAFANA_USERNAME env var must be set")
		}
		grafanaToken = os.Getenv("GRAFANA_TOKEN")
		if grafanaToken == "" {
			log.Fatal().Msg("GRAFANA_TOKEN env var must be set")
		}
	}

	_, err = testnet.CreateGenesisNodes(2, appVersion, 10000000)
	if err != nil {
		panic(err)
	}

	fullNodes, err := testnet.CreateNodes(2, appVersion, 0)
	if err != nil {
		panic(err)
	}

	err = testnet.Setup()
	if err != nil {
		panic(err)
	}

	err = testnet.Start()
	if err != nil {
		panic(err)
	}

	// Note:
	// With the current implementation you need to have the consensusNode and the trustedPeers already running
	// before you can start a node connected to them.
	// Future work should support the same approach as for the consensus part.

	bridge1, err := testnet.CreateDaNode(bridge, nodeVersion, fullNodes[0], []*DaNode{})
	if err != nil {
		panic(err)
	}
	err = bridge1.Init()
	if err != nil {
		panic(err)
	}
	err = bridge1.Start()
	if err != nil {
		panic(err)
	}
	bridge2, err := testnet.CreateDaNode(bridge, nodeVersion, fullNodes[1], []*DaNode{})
	if err != nil {
		panic(err)
	}
	err = bridge2.Init()
	if err != nil {
		panic(err)
	}
	err = bridge2.Start()
	if err != nil {
		panic(err)
	}

	full1, err := testnet.CreateDaNode(full, nodeVersion, fullNodes[0], []*DaNode{bridge1})
	if err != nil {
		panic(err)
	}
	err = full1.Init()
	if err != nil {
		panic(err)
	}
	err = full1.Start()
	if err != nil {
		panic(err)
	}

	full2, err := testnet.CreateDaNode(full, nodeVersion, fullNodes[1], []*DaNode{bridge2})
	if err != nil {
		panic(err)
	}
	err = full2.Init()
	if err != nil {
		panic(err)
	}
	err = full2.Start()
	if err != nil {
		panic(err)
	}

	light1, err := testnet.CreateDaNode(light, nodeVersion, fullNodes[0], []*DaNode{bridge1, bridge2, full1, full2})
	if err != nil {
		panic(err)
	}
	err = light1.Init()
	if err != nil {
		panic(err)
	}
	err = light1.Start()
	if err != nil {
		panic(err)
	}
	light2, err := testnet.CreateDaNode(light, nodeVersion, fullNodes[1], []*DaNode{bridge1, bridge2, full1, full2})
	if err != nil {
		panic(err)
	}
	err = light2.Init()
	if err != nil {
		panic(err)
	}
	err = light2.Start()
	if err != nil {
		panic(err)
	}

	//_, mnemomic, err := testnet.CreateGenesisAccount("txsim", 1e12)
	//if err != nil {
	//	panic(err)
	//}
	//
	//err = testnet.CreateTxSim(mnemomic, txSimVersion, 15*time.Second, []int{50000, 100000}, 100, 1, 50, 100)
	//if err != nil {
	//	panic(err)
	//}
}
