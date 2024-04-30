package knuu

import (
	"context"
	"fmt"

	"github.com/celestiaorg/knuu/pkg/k8s"
	v1 "k8s.io/api/core/v1"
)

// Preloader is a struct that contains the list of preloaded images.
// A preloader makes sure that the images are preloaded before the test suite starts.
// Hint: If you use a Preloader per test suite, you can save resources
type Preloader struct {
	K8sName string   `json:"k8sName"`
	Images  []string `json:"images"`
}

// NewPreloader creates a new preloader
func NewPreloader() (*Preloader, error) {
	k8sName, err := generateK8sName("knuu-preloader")
	if err != nil {
		return nil, ErrGeneratingK8sNameForPreloader.Wrap(err)
	}
	return &Preloader{
		K8sName: k8sName,
		Images:  []string{},
	}, nil
}

// Images returns the list of preloaded images
func (p *Preloader) GetImages() []string {
	return p.Images
}

// AddImage adds an image to the list of preloaded images
func (p *Preloader) AddImage(image string) error {
	// dont add duplicates
	for _, v := range p.Images {
		if v == image {
			return nil
		}
	}
	p.Images = append(p.Images, image)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return p.preloadImages(ctx)
}

// RemoveImage removes an image from the list of preloaded images
func (p *Preloader) RemoveImage(image string) error {
	for i, v := range p.Images {
		if v == image {
			p.Images = append(p.Images[:i], p.Images[i+1:]...)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return p.preloadImages(ctx)
}

// EmptyImages empties the list of preloaded images
func (p *Preloader) EmptyImages() error {
	p.Images = []string{}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return p.preloadImages(ctx)
}

// preloadImages preloads all images in the list of preloaded images
func (p *Preloader) preloadImages(ctx context.Context) error {
	// delete the daemonset if no images are preloaded
	if len(p.Images) == 0 {
		return k8sClient.DeleteDaemonSet(ctx, p.K8sName)
	}
	var initContainers []v1.Container

	for i, image := range p.Images {
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
		"app":                          p.K8sName,
		"k8s.kubernetes.io/managed-by": "knuu",
		"knuu.sh/scope":                k8s.SanitizeName(testScope),
		"knuu.sh/test-started":         startTime,
	}

	exists, err := k8sClient.DaemonSetExists(ctx, p.K8sName)
	if err != nil {
		return err
	}

	// update the daemonset if it already exists
	if exists {
		_, err = k8sClient.UpdateDaemonSet(ctx, p.K8sName, labels, initContainers, containers)
		return err
	}

	// create the daemonset if it doesn't exist
	_, err = k8sClient.CreateDaemonSet(ctx, p.K8sName, labels, initContainers, containers)
	return err
}
