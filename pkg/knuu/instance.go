// Package knuu provides the core functionality of knuu.
package knuu

import (
	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/preloader"
)

func (k *Knuu) NewInstance(name string) (*instance.Instance, error) {
	return instance.New(name, k.SystemDependencies)
}

func (k *Knuu) NewPreloader(name string) (*preloader.Preloader, error) {
	return preloader.New(name, k.SystemDependencies)
}
