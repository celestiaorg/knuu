package system

import (
	"sync"

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
	Scope        string
	StartTime    string
	instancesMap sync.Map
}

func (s *SystemDependencies) AddInstanceName(name string) {
	s.instancesMap.Store(name, struct{}{})
}

func (s *SystemDependencies) HasInstanceName(name string) bool {
	_, exists := s.instancesMap.Load(name)
	return exists
}

func (s *SystemDependencies) RemoveInstanceName(name string) {
	s.instancesMap.Delete(name)
}
