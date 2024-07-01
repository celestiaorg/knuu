package preloader

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"

	"github.com/celestiaorg/knuu/pkg/names"
	"github.com/celestiaorg/knuu/pkg/system"
)

const (
	preloaderName        = "knuu-preloader"
	managedByLabel       = "knuu"
	pauseContainerImage  = "k8s.gcr.io/pause"
	preloaderCommand     = "/bin/sh"
	preloaderCommandArgs = "-c"
	preloaderCommandExit = "exit 0"

	pauseContainerName = "pause-container"

	labelApp         = "app"
	labelManagedBy   = "k8s.kubernetes.io/managed-by"
	labelScope       = "knuu.sh/scope"
	labelTestStarted = "knuu.sh/test-started"
)

// Preloader is a struct that contains the list of preloaded images.
// A preloader makes sure that the images are preloaded before the test suite starts.
// Hint: If you use a Preloader per test suite, you can save resources
type Preloader struct {
	K8sName string   `json:"k8sName"`
	Images  []string `json:"images"`
	system.SystemDependencies
}

// New creates a new preloader
func New(sysDeps system.SystemDependencies) (*Preloader, error) {
	k8sName, err := names.NewRandomK8(preloaderName)
	if err != nil {
		return nil, ErrGeneratingK8sNameForPreloader.Wrap(err)
	}
	return &Preloader{
		K8sName:            k8sName,
		Images:             []string{},
		SystemDependencies: sysDeps,
	}, nil
}

// Images returns the list of preloaded images
func (p *Preloader) GetImages() []string {
	return p.Images
}

// AddImage adds an image to the list of preloaded images
func (p *Preloader) AddImage(ctx context.Context, image string) error {
	// don't add duplicates
	for _, v := range p.Images {
		if v == image {
			return nil
		}
	}
	p.Images = append(p.Images, image)

	return p.preloadImages(ctx)
}

// RemoveImage removes an image from the list of preloaded images
func (p *Preloader) RemoveImage(ctx context.Context, image string) error {
	for i, v := range p.Images {
		if v == image {
			p.Images = append(p.Images[:i], p.Images[i+1:]...)
		}
	}

	return p.preloadImages(ctx)
}

// EmptyImages empties the list of preloaded images
func (p *Preloader) EmptyImages(ctx context.Context) error {
	p.Images = []string{}
	return p.preloadImages(ctx)
}

// preloadImages preloads all images in the list of preloaded images
func (p *Preloader) preloadImages(ctx context.Context) error {
	// delete the daemonset if no images are preloaded
	if len(p.Images) == 0 {
		return p.K8sClient.DeleteDaemonSet(ctx, p.K8sName)
	}
	var initContainers []v1.Container

	for i, image := range p.Images {
		initContainers = append(initContainers, v1.Container{
			Name:  fmt.Sprintf("image%d-preloader", i),
			Image: image,
			Command: []string{
				preloaderCommand,
				preloaderCommandArgs,
				preloaderCommandExit,
			},
		})
	}

	var containers []v1.Container

	containers = append(containers, v1.Container{
		Name:  pauseContainerName,
		Image: pauseContainerImage,
	})

	labels := map[string]string{
		labelApp:         p.K8sName,
		labelManagedBy:   managedByLabel,
		labelScope:       p.TestScope,
		labelTestStarted: p.StartTime,
	}

	exists, err := p.K8sClient.DaemonSetExists(ctx, p.K8sName)
	if err != nil {
		return err
	}

	// update the daemonset if it already exists
	if exists {
		_, err = p.K8sClient.UpdateDaemonSet(ctx, p.K8sName, labels, initContainers, containers)
		return err
	}

	// create the daemonset if it doesn't exist
	_, err = p.K8sClient.CreateDaemonSet(ctx, p.K8sName, labels, initContainers, containers)
	return err
}
