// Package knuu provides the core functionality of knuu.
package knuu

import (
	"errors"
	"os"

	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/containers/buildah"
	"github.com/containers/storage/pkg/unshare"
	"github.com/sirupsen/logrus"
)

// Initialize initializes knuu
func Initialize() error {

	if buildah.InitReexec() {
		return errors.New("InitReexec triggered re-exec")
	}
	unshare.MaybeReexecUsingUserNamespace(false)

	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}

	err := k8s.Initialize()
    if err != nil {
        return err
    }

	return nil
}
