package system

import (
	"github.com/sirupsen/logrus"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/minio"
	"github.com/celestiaorg/knuu/pkg/traefik"
)

type SystemDependencies struct {
	ImageBuilder builder.Builder
	K8sClient    k8s.KubeManager
	MinioClient  *minio.Minio
	Logger       *logrus.Logger
	Proxy        *traefik.Traefik
	TestScope    string
	StartTime    string
}
