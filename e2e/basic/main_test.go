package basic

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/celestiaorg/knuu/pkg/knuu"
)

func TestMain(m *testing.M) {
	err := knuu.Initialize()
	if err != nil {
		logrus.Fatalf("error initializing knuu: %v", err)
	}
	logrus.Infof("Scope: %s", knuu.Scope())
	exitVal := m.Run()
	os.Exit(exitVal)
}
