package log

import (
	"os"
	"path"
	"runtime"
	"strconv"

	"github.com/sirupsen/logrus"
)

const envLogLevel = "LOG_LEVEL"

func DefaultLogger() *logrus.Logger {
	logger := logrus.New()

	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			var (
				filename  = path.Base(f.File)
				directory = path.Base(path.Dir(f.File))
			)
			return "", directory + "/" + filename + ":" + strconv.Itoa(f.Line)
		},
	})

	// Enable reporting the file and line
	logger.SetReportCaller(true)

	if customLevel := os.Getenv(envLogLevel); customLevel != "" {
		err := logger.Level.UnmarshalText([]byte(customLevel))
		if err != nil {
			logger.WithFields(logrus.Fields{
				"env_log_level": envLogLevel,
				"error":         err,
			}).
				Warn("Failed to parse env var LOG_LEVEL, defaulting to INFO")
		}
	}
	logger.WithFields(logrus.Fields{
		"env_log_level": envLogLevel,
		"log_level":     logger.GetLevel(),
	}).Info("Current log level")

	return logger
}
