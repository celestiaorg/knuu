// Package container provides utility functions for working with containers.
package container

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containers/buildah"
	"github.com/containers/image/v5/docker"
	is "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/sirupsen/logrus"
)

// BuilderFactory is responsible for creating new instances of buildah.Builder
type BuilderFactory struct {
	ctx     context.Context
	builder *buildah.Builder
	store   storage.Store
}

// NewBuilderFactory creates a new instance of BuilderFactory.
func NewBuilderFactory(imageName string) (*BuilderFactory, error) {
	ctx, _ := context.WithCancel(context.Background())
	if imageName == "" {
		return nil, fmt.Errorf("image name cannot be empty")
	}

	logrus.Debugf("Creating new Buildah builder for image '%s'", imageName)

	storeOpts, err := storage.DefaultStoreOptionsAutoDetectUID()
	if err != nil {
		return nil, fmt.Errorf("failed to get default storage options: %w", err)
	}

	buildStore, err := storage.GetStore(storeOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage store: %w", err)
	}

	builderOpts := buildah.BuilderOptions{
		FromImage: imageName,
	}

	builder, err := buildah.NewBuilder(ctx, buildStore, builderOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create new builder: %w", err)
	}

	logrus.Debugf("Successfully created new Buildah builder for image '%s'", imageName)

	return &BuilderFactory{
		ctx:     ctx,
		builder: builder,
		store:   buildStore,
	}, nil
}

// ExecuteCmdInBuilder runs the provided command in the context of the given builder.
// It returns the command's output or any error encountered.
func (f *BuilderFactory) ExecuteCmdInBuilder(command []string) (string, error) {
	var stdout, stderr bytes.Buffer

	runOpts := buildah.RunOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	}

	if err := f.builder.Run(command, runOpts); err != nil {
		errMessage := fmt.Errorf("running command: %w.", err)
		if stderr.String() != "" {
			errMessage = fmt.Errorf("%s Stderr: %s", errMessage, stderr.String())
		}
		return "", errMessage
	}

	logrus.Debugf("Successfully executed command '%v' in builder", command)
	return stdout.String(), nil
}

// AddFileToBuilder adds a file from the source path to the destination path in the image, with the specified ownership.
func (f *BuilderFactory) AddFileToBuilder(srcPath, destPath, chown string) error {
	addOpts := buildah.AddAndCopyOptions{Chown: chown}

	if err := f.builder.Add(destPath, false, addOpts, srcPath); err != nil {
		return fmt.Errorf("failed to add file to image: %w", err)
	}

	logrus.Debugf("file %s added to image at %s", srcPath, destPath)

	return nil
}

// ReadFileFromBuilder reads a file from the given builder's mount point.
// It returns the file's content or any error encountered.
func (f *BuilderFactory) ReadFileFromBuilder(filePath string) ([]byte, error) {
	// Mount builder's mount point.
	mountPoint, err := f.builder.Mount("")
	if err != nil {
		return nil, fmt.Errorf("failed to mount build container: %w", err)
	}

	// Create full path using provided file path and the mount point.
	fullPath := filepath.Join(mountPoint, filePath)

	// Read file content from the full path.
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file from build container: %w", err)
	}

	return content, nil
}

// SetEnvVar sets the value of an environment variable in the builder.
func (f *BuilderFactory) SetEnvVar(name, value string) error {
	// Set and log the value of the environment variable
	f.builder.SetEnv(name, value)
	logrus.Debugf("Set environment variable %s=%s", name, value)

	return nil
}

// PushBuilderImage pushes the image from the given builder to a registry.
// The image is identified by the provided name.
func (f *BuilderFactory) PushBuilderImage(imageName string) error {
	// Parse image reference
	imgRef, err := is.Transport.ParseStoreReference(f.store, imageName)
	if err != nil {
		return fmt.Errorf("failed to parse image reference: %w", err)
	}

	// Commit and get image ID
	imgID, _, _, err := f.builder.Commit(f.ctx, imgRef, buildah.CommitOptions{})
	if err != nil {
		return fmt.Errorf("failed to commit image: %w", err)
	}
	logrus.Debugf("Committed image '%s' with ID '%s'", imageName, imgID)

	// Parse destination reference
	dest, err := docker.ParseReference("//" + imageName)
	if err != nil {
		return fmt.Errorf("failed to parse destination reference: %w", err)
	}

	// Push image with options
	pushOpts := buildah.PushOptions{
		Store:         f.store,
		SystemContext: &types.SystemContext{},
	}

	ref, _, err := buildah.Push(f.ctx, imageName, dest, pushOpts)
	if err != nil {
		return fmt.Errorf("failed to push image: %w", err)
	}
	logrus.Debugf("Pushed image '%s' with ref '%s'", imageName, ref)

	return nil
}
