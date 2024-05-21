package system

import (
	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/minio"
	"github.com/sirupsen/logrus"
)

type SystemDependencies struct {
	ImageBuilder builder.Builder
	K8sCli       *k8s.Client
	MinioCli     *minio.Minio
	Logger       *logrus.Logger
	TestScope    string
	StartTime    string
}
