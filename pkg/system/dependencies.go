package system

import (
	"github.com/sirupsen/logrus"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/minio"
	"github.com/celestiaorg/knuu/pkg/traefik"
)

type SystemDependencies struct {
	ImageBuilder     builder.Builder
	K8sClient        k8s.KubeManager
	MinioClient      *minio.Minio
	Logger           *logrus.Logger
	Proxy            *traefik.Traefik
	enabledChaosMesh bool
	TestScope        string
	StartTime        string
}

func (s *SystemDependencies) SetChaosMeshEnabled(enabled bool) {
	s.enabledChaosMesh = enabled
}

func (s *SystemDependencies) IsChaosMeshEnabled() bool {
	return s.enabledChaosMesh
}
