package celestia_app

import (
	"os"
	"testing"

	"github.com/celestiaorg/knuu/pkg/knuu"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	err := knuu.Initialize()
	if err != nil {
		logrus.Fatalf("error initializing knuu: %v", err)
	}
	logrus.Infof("Scope: %s", knuu.Scope())
	prepareInstances(m)
	exitVal := m.Run()
	os.Exit(exitVal)
}

var Instances = map[string]*knuu.Instance{}

func prepareInstances(m *testing.M) {
	validator, err := knuu.NewInstance("validator")
	if err != nil {
		logrus.Fatalf("Error creating instance '%v':", err)
	}
	err = validator.SetImage("ghcr.io/celestiaorg/celestia-app:v1.7.0")
	if err != nil {
		logrus.Fatalf("Error setting image: %v", err)
	}
	err = validator.AddPortTCP(26656)
	if err != nil {
		logrus.Fatalf("Error adding port: %v", err)
	}
	err = validator.AddPortTCP(26657)
	if err != nil {
		logrus.Fatalf("Error adding port: %v", err)
	}
	err = validator.AddFile("resources/genesis.sh", "/home/celestia/genesis.sh", "10001:10001")
	if err != nil {
		logrus.Fatalf("Error adding file: %v", err)
	}
	_, err = validator.ExecuteCommand("/bin/sh", "/home/celestia/genesis.sh")
	if err != nil {
		logrus.Fatalf("Error executing command: %v", err)
	}
	err = validator.SetArgs("start", "--home=/home/celestia", "--rpc.laddr=tcp://0.0.0.0:26657")
	if err != nil {
		logrus.Fatalf("Error setting args: %v", err)
	}
	err = validator.SetMemory("200Mi", "200Mi")
	if err != nil {
		logrus.Fatalf("Error setting memory: %v", err)
	}
	err = validator.SetCPU("300m")
	err = validator.Commit()
	if err != nil {
		logrus.Fatalf("Error committing instance: %v", err)
	}

	Instances["validator"] = validator

	full, err := knuu.NewInstance("full")
	if err != nil {
		logrus.Fatalf("Error creating instance '%v':", err)
	}
	err = full.SetImage("ghcr.io/celestiaorg/celestia-app:v1.7.0")
	if err != nil {
		logrus.Fatalf("Error setting image: %v", err)
	}
	err = full.AddPortTCP(26656)
	if err != nil {
		logrus.Fatalf("Error adding port: %v", err)
	}
	err = full.AddPortTCP(26657)
	if err != nil {
		logrus.Fatalf("Error adding port: %v", err)
	}
	genesis, err := validator.GetFileBytes("/home/celestia/config/genesis.json")
	if err != nil {
		logrus.Fatalf("Error getting genesis: %v", err)
	}
	err = full.AddFileBytes(genesis, "/home/celestia/config/genesis.json", "10001:10001")
	if err != nil {
		logrus.Fatalf("Error adding file: %v", err)
	}
	err = full.SetMemory("200Mi", "200Mi")
	if err != nil {
		logrus.Fatalf("Error setting memory: %v", err)
	}
	err = full.SetCPU("300m")
	err = full.Commit()
	if err != nil {
		logrus.Fatalf("Error committing instance: %v", err)
	}

	Instances["full"] = full
}

func forwardBitTwisterPort(t *testing.T, i *knuu.Instance) {
	fwdBtPort, err := i.PortForwardTCP(i.BitTwister.Port())
	require.NoError(t, err, "Error port forwarding")
	i.BitTwister.SetPort(fwdBtPort)
	i.BitTwister.SetNewClientByURL("http://localhost")
	t.Logf("BitTwister listening on http://localhost:%d", fwdBtPort)
}
