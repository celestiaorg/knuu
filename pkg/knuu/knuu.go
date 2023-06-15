// Package knuu provides the core functionality of knuu.
package knuu

import (
	"fmt"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

// Identifier is the identifier of the current knuu instance
var identifier string
var startTime string

// Initialize initializes knuu
// Deprecated: Use InitializeWithIdentifier instead
func Initialize() error {

	t := time.Now()
	identifier = fmt.Sprintf("%s_%03d", t.Format("20060102_150405"), t.Nanosecond()/1e6)
	return InitializeWithIdentifier(identifier)
}

// Identifier returns the identifier of the current knuu instance
func Identifier() string {
	return identifier
}

// InitializeWithIdentifier initializes knuu with a unique identifier
func InitializeWithIdentifier(uniqueIdentifier string) error {
	if uniqueIdentifier == "" {
		return fmt.Errorf("cannot initialize knuu with empty identifier")
	}
	identifier = uniqueIdentifier

	t := time.Now()
	startTime = fmt.Sprintf("%s_%03d", t.Format("20060102_150405"), t.Nanosecond()/1e6)

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

// IsInitialized returns true if knuu is initialized, and false otherwise
func IsInitialized() bool {
	return k8s.IsInitialized()
}
