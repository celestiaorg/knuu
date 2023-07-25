package knuu

import (
	"fmt"
	"github.com/celestiaorg/knuu/pkg/k8s"
	v1 "k8s.io/api/core/v1"
)

// Preloader is a struct that contains the list of preloaded images.
// A preloader makes sure that the images are preloaded before the test suite starts.
// Hint: If you use a Preloader per test suite, you can save resources
type Preloader struct {
	k8sName string   `json:"k8sName"`
	images  []string `json:"images"`
}

// NewPreloader creates a new preloader
func NewPreloader() (*Preloader, error) {
	k8sName, err := generateK8sName("knuu-preloader")
	if err != nil {
		return nil, fmt.Errorf("error generating k8s name for preloader: %w", err)
	}
	return &Preloader{
		k8sName: k8sName,
		images:  []string{},
	}, nil
}

// Images returns the list of preloaded images
func (p *Preloader) Images() []string {
	return p.images
}

// AddImage adds an image to the list of preloaded images
func (p *Preloader) AddImage(image string) error {
	// dont add duplicates
	for _, v := range p.images {
		if v == image {
			return nil
		}
	}
	p.images = append(p.images, image)
	return p.preloadImages()
}

// RemoveImage removes an image from the list of preloaded images
func (p *Preloader) RemoveImage(image string) error {
	for i, v := range p.images {
		if v == image {
			p.images = append(p.images[:i], p.images[i+1:]...)
		}
	}
	return p.preloadImages()
}

// EmptyImages empties the list of preloaded images
func (p *Preloader) EmptyImages() error {
	p.images = []string{}
	return p.preloadImages()
}

// preloadImages preloads all images in the list of preloaded images
func (p *Preloader) preloadImages() error {
	// delete the daemonset if no images are preloaded
	if len(p.images) == 0 {
		return k8s.DeleteDaemonSet(k8s.Namespace(), p.k8sName)
	}
	var initContainers []v1.Container

	for i, image := range p.images {
		initContainers = append(initContainers, v1.Container{
			Name:  fmt.Sprintf("image%d-preloader", i),
			Image: image,
			Command: []string{
				"/bin/sh",
				"-c",
				"exit 0",
			},
		})
	}

	var containers []v1.Container

	containers = append(containers, v1.Container{
		Name:  "pause-container",
		Image: "k8s.gcr.io/pause",
	})

	labels := map[string]string{
		"app":                          p.k8sName,
		"k8s.kubernetes.io/managed-by": "knuu",
		"knuu.sh/test-run-id":                  identifier,
		"knuu.sh/test-started":                 startTime,
	}

	exists, err := k8s.DaemonSetExists(k8s.Namespace(), p.k8sName)
	if err != nil {
		return err
	}

	// update the daemonset if it already exists
	if exists {
		_, err = k8s.UpdateDaemonSet(k8s.Namespace(), p.k8sName, labels, initContainers, containers)
		return err
	}

	// create the daemonset if it doesn't exist
	_, err = k8s.CreateDaemonSet(k8s.Namespace(), p.k8sName, labels, initContainers, containers)
	return err
}
