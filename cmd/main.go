package main

import (
	"github.com/celestiaorg/knuu/cmd/root"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := root.Execute(); err != nil {
		logrus.WithError(err).Fatal("failed to execute command")
	}
}
