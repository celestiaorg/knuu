package system

import (
	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/minio"
	"github.com/celestiaorg/knuu/pkg/traefik"
	"github.com/sirupsen/logrus"
)

type SystemDependencies struct {
	ImageBuilder builder.Builder
	K8sCli       k8s.KubeManager
	MinioCli     *minio.Minio
	Logger       *logrus.Logger
	Proxy        *traefik.Traefik
	TestScope    string
	StartTime    string
}
