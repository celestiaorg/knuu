package log

import (
	"os"
	"path"
	"runtime"
	"strconv"

	"github.com/sirupsen/logrus"
)

func DefaultLogger() *logrus.Logger {
	logger := logrus.New()

	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			directory := path.Base(path.Dir(f.File))
			return "", directory + "/" + filename + ":" + strconv.Itoa(f.Line)
		},
	})

	// Enable reporting the file and line
	logger.SetReportCaller(true)

	customLevel := os.Getenv("LOG_LEVEL")
	if customLevel != "" {
		err := logger.Level.UnmarshalText([]byte(customLevel))
		if err != nil {
			logger.Warnf("Failed to parse LOG_LEVEL: %v, defaulting to INFO", err)
		}
	}
	logger.Info("LOG_LEVEL: ", logger.GetLevel())

	return logger
}
