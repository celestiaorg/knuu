package basic

import (
	"os"
	"testing"

	"github.com/celestiaorg/knuu/pkg/knuu"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	err := knuu.Initialize()
	if err != nil {
		logrus.Fatalf("error initializing knuu: %v", err)
	}
	logrus.Infof("Scope: %s", knuu.Scope())

	knuu.HandleStopSignal()

	exitVal := m.Run()
	os.Exit(exitVal)
}
