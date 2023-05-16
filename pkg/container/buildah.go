// Package container provides utility functions for working with buildah containers.
package container

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/containers/buildah"
	"github.com/containers/image/v5/docker"
	is "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/sirupsen/logrus"
)

// NewBuilder initiates a new buildah builder for the given image.
// It returns the new builder, the storage.Store object and any error encountered.
func NewBuilder(ctx context.Context, imageName string) (*buildah.Builder, storage.Store, error) {
	storeOpts, err := storage.DefaultStoreOptionsAutoDetectUID()
	if err != nil {
		return nil, nil, fmt.Errorf("getting default storage options: %w", err)
	}

	buildStore, err := storage.GetStore(storeOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("creating storage store: %w", err)
	}

	builderOpts := buildah.BuilderOptions{
		FromImage: imageName,
	}

	builder, err := buildah.NewBuilder(ctx, buildStore, builderOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("creating new builder: %w", err)
	}

	return builder, buildStore, nil
}

// ExecuteCmdInBuilder runs the provided command in the context of the given builder.
// It returns the command's output or any error encountered.
func ExecuteCmdInBuilder(builder *buildah.Builder, command []string) (string, error) {
	var stdout, stderr bytes.Buffer

	runOpts := buildah.RunOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	}

	if err := builder.Run(command, runOpts); err != nil {
		if stderr.String() != "" {
			return "", fmt.Errorf("running command: %w. Stderr: %s", err, stderr.String())
		}
		return "", fmt.Errorf("running command: %w", err)
	}

	return stdout.String(), nil
}

// AddFileToBuilder adds a file from the source path to the destination path in the image, with the specified ownership.
func AddFileToBuilder(builder *buildah.Builder, srcPath, destPath, chown string) error {
	addOpts := buildah.AddAndCopyOptions{Chown: chown}

	if err := builder.Add(destPath, false, addOpts, srcPath); err != nil {
		return fmt.Errorf("adding file to image: %w", err)
	}

	return nil
}

// ReadFileFromBuilder reads a file from the given builder's mount point.
// It returns the file's content or any error encountered.
func ReadFileFromBuilder(builder *buildah.Builder, filePath string) ([]byte, error) {
	mountPoint, err := builder.Mount("")
	if err != nil {
		return nil, fmt.Errorf("mounting build container: %w", err)
	}
	defer builder.Unmount()

	fullPath := filepath.Join(mountPoint, filePath)
	content, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("reading file from build container: %w", err)
	}

	return content, nil
}

// SetEnvVar sets an environment variable in the given builder.
func SetEnvVar(builder *buildah.Builder, name, value string) {
	builder.SetEnv(name, value)
}

// PushBuilderImage pushes the image from the given builder to a registry.
// The image is identified by the provided name.
func PushBuilderImage(ctx context.Context, builder *buildah.Builder, buildStore storage.Store, imageName string) error {
	imgRef, err := is.Transport.ParseStoreReference(buildStore, imageName)
	if err != nil {
		return fmt.Errorf("parsing image reference: %w", err)
	}

	imgID, _, _, err := builder.Commit(ctx, imgRef, buildah.CommitOptions{})
	if err != nil {
		return fmt.Errorf("committing image: %w", err)
	}

	logrus.Debugf("Committed image '%s' with ID '%s'", imageName, imgID)

	dest, err := docker.ParseReference("//" + imageName)
	if err != nil {
		return fmt.Errorf("parsing destination reference: %w", err)
	}

	pushOpts := buildah.PushOptions{
		Store:         buildStore,
		SystemContext: &types.SystemContext{},
	}

	ref, _, err := buildah.Push(ctx, imageName, dest, pushOpts)
	if err != nil {
		return fmt.Errorf("pushing image: %w", err)
	}

	logrus.Debugf("Pushed image '%s' with ref '%s'", imageName, ref)

	return nil
}
