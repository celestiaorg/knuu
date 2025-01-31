package main

import (
	"github.com/sirupsen/logrus"

	"github.com/celestiaorg/knuu/cmd/root"
)

func main() {
	if err := root.Execute(); err != nil {
		logrus.WithError(err).Fatal("failed to execute command")
	}
}
