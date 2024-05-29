/*
* This file is deprecated.
* Please use the new package knuu instead.
* This file keeps the old functionality of knuu for backward compatibility.
* A global variable is defined, tmpKnuu, which is used to hold the knuu instance.
 */

package knuu

import (
	"context"

	"github.com/celestiaorg/knuu/pkg/preloader"
)

type Preloader struct {
	preloader.Preloader
}

// Deprecated: Use the new package knuu instead.
func NewPreloader() (*Preloader, error) {
	p, err := tmpKnuu.NewPreloader()
	if err != nil {
		return nil, err
	}
	return &Preloader{Preloader: *p}, nil
}

// Deprecated: Use the new package knuu instead.
func (p *Preloader) AddImage(image string) error {
	return p.Preloader.AddImage(context.Background(), image)
}

// Deprecated: Use the new package knuu instead.
func (p *Preloader) RemoveImage(image string) error {
	return p.Preloader.RemoveImage(context.Background(), image)
}

// Deprecated: Use the new package knuu instead.
func (p *Preloader) EmptyImages() error {
	return p.Preloader.EmptyImages(context.Background())
}
