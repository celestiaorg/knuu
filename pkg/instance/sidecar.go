package instance

import (
	"context"

	"github.com/celestiaorg/knuu/pkg/system"
)

type SidecarManager interface {
	Initialize(ctx context.Context, sysDeps system.SystemDependencies) error
	Instance() *Instance
	PreStart(ctx context.Context) error
	CloneWithSuffix(suffix string) SidecarManager
}
