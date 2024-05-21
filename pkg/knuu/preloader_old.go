// Package knuu provides the core functionality of knuu.
package knuu

import (
	"context"

	"github.com/celestiaorg/knuu/pkg/preloader"
)

type Preloader struct {
	preloader.Preloader
}

func NewPreloader() (*Preloader, error) {
	p, err := tmpKnuu.NewPreloader()
	if err != nil {
		return nil, err
	}
	return &Preloader{Preloader: *p}, nil
}

func (p *Preloader) AddImage(image string) error {
	return p.Preloader.AddImage(context.Background(), image)
}

func (p *Preloader) RemoveImage(image string) error {
	return p.Preloader.RemoveImage(context.Background(), image)
}

func (p *Preloader) EmptyImages() error {
	return p.Preloader.EmptyImages(context.Background())
}
