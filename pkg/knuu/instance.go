// Package knuu provides the core functionality of knuu.
package knuu

import (
	"context"

	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/preloader"
)

func (k *Knuu) NewInstance(name string) (*instance.Instance, error) {
	return instance.New(name, k.SystemDependencies)
}

func (k *Knuu) NewExecutor(ctx context.Context) (*instance.Executor, error) {
	return instance.NewExecutor(ctx, k.SystemDependencies)
}

func (k *Knuu) NewPreloader() (*preloader.Preloader, error) {
	return preloader.New(k.SystemDependencies)
}
