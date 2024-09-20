// Package container provides utilities for interacting with containers.
package container

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/celestiaorg/knuu/pkg/builder"
)

// BuilderFactory is responsible for creating new instances of buildah.Builder
type BuilderFactory struct {
	imageNameFrom          string
	imageNameTo            string
	imageBuilder           builder.Builder
	dockerFileInstructions []string
	buildContext           string
}

// NewBuilderFactory creates a new instance of BuilderFactory.
func NewBuilderFactory(imageName, buildContext string, imageBuilder builder.Builder) (*BuilderFactory, error) {
	if err := os.MkdirAll(buildContext, 0755); err != nil {
		return nil, ErrFailedToCreateContextDir.Wrap(err)
	}

	return &BuilderFactory{
		imageNameFrom:          imageName,
		dockerFileInstructions: []string{"FROM " + imageName},
		buildContext:           buildContext,
		imageBuilder:           imageBuilder,
	}, nil
}

// ImageNameFrom returns the name of the image from which the builder is created.
func (f *BuilderFactory) ImageNameFrom() string {
	return f.imageNameFrom
}

// AddCmdToBuilder adds the provided command to be run in the context of the builder.
func (f *BuilderFactory) AddCmdToBuilder(command []string) {
	f.dockerFileInstructions = append(f.dockerFileInstructions, "RUN "+strings.Join(command, " "))
}

// AddToBuilder adds a file from the source path to the destination path in the image, with the specified ownership.
func (f *BuilderFactory) AddToBuilder(srcPath, destPath, chown string) {
	f.dockerFileInstructions = append(f.dockerFileInstructions, "ADD --chown="+chown+" "+srcPath+" "+destPath)
}

// SetEnvVar sets the value of an environment variable in the builder.
func (f *BuilderFactory) SetEnvVar(name, value string) {
	f.dockerFileInstructions = append(f.dockerFileInstructions, "ENV "+name+"="+value)
}

// SetUser sets the user in the builder.
func (f *BuilderFactory) SetUser(user string) {
	f.dockerFileInstructions = append(f.dockerFileInstructions, "USER "+user)
}

// Changed returns true if the builder has been modified, false otherwise.
func (f *BuilderFactory) Changed() bool {
	return len(f.dockerFileInstructions) > 1
}

// PushBuilderImage pushes the image from the given builder to a registry.
// The image is identified by the provided name.
func (f *BuilderFactory) PushBuilderImage(ctx context.Context, imageName string) error {
	if !f.Changed() {
		logrus.Debugf("No changes made to image %s, skipping push", f.imageNameFrom)
		return nil
	}

	f.imageNameTo = imageName

	dockerFilePath := filepath.Join(f.buildContext, "Dockerfile")
	// create path if it does not exist
	if _, err := os.Stat(f.buildContext); os.IsNotExist(err) {
		err = os.MkdirAll(f.buildContext, 0755)
		if err != nil {
			return ErrFailedToCreateContextDir.Wrap(err)
		}
	}

	dockerFile := strings.Join(f.dockerFileInstructions, "\n")
	err := os.WriteFile(dockerFilePath, []byte(dockerFile), 0644)
	if err != nil {
		return ErrFailedToWriteDockerfile.Wrap(err)
	}

	logs, err := f.imageBuilder.Build(ctx, &builder.BuilderOptions{
		ImageName:    f.imageNameTo,
		Destination:  f.imageNameTo, // in docker the image name and destination are the same
		BuildContext: builder.DirContext{Path: f.buildContext}.BuildContext(),
	})

	qStatus := logrus.TextFormatter{}.DisableQuote
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableQuote: true,
	})
	logrus.Debug("build logs: ", logs)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableQuote: qStatus,
	})

	return err
}

// BuildImageFromGitRepo builds an image from the given git repository and
// pushes it to a registry. The image is identified by the provided name.
func (f *BuilderFactory) BuildImageFromGitRepo(ctx context.Context, gitCtx builder.GitContext, imageName string) error {
	buildCtx, err := gitCtx.BuildContext()
	if err != nil {
		return ErrFailedToGetBuildContext.Wrap(err)
	}

	f.imageNameTo = imageName

	cOpts := &builder.CacheOptions{}
	cOpts, err = cOpts.Default(buildCtx)
	if err != nil {
		return ErrFailedToGetDefaultCacheOptions.Wrap(err)
	}

	logrus.Debugf("Building image %s from git repo %s", imageName, gitCtx.Repo)

	logs, err := f.imageBuilder.Build(ctx, &builder.BuilderOptions{
		ImageName:    imageName,
		Destination:  imageName,
		BuildContext: buildCtx,
		Cache:        cOpts,
	})

	qStatus := logrus.TextFormatter{}.DisableQuote
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableQuote: true,
	})

	logrus.Debug("build logs: ", logs)

	logrus.SetFormatter(&logrus.TextFormatter{
		DisableQuote: qStatus,
	})
	return err
}

// GenerateImageHash creates a hash value based on the contents of the Dockerfile instructions and all files in the build context.
func (f *BuilderFactory) GenerateImageHash() (string, error) {
	hasher := sha256.New()

	// Hash Dockerfile content
	dockerFileContent := strings.Join(f.dockerFileInstructions, "\n")
	_, err := hasher.Write([]byte(dockerFileContent))
	if err != nil {
		return "", ErrHashingDockerfile.Wrap(err)
	}

	// Hash contents of all files in the build context
	err = filepath.Walk(f.buildContext, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileContent, err := os.ReadFile(path)
			if err != nil {
				return ErrReadingFile.WithParams(path).Wrap(err)
			}
			_, err = hasher.Write(fileContent)
			if err != nil {
				return ErrHashingFile.WithParams(path).Wrap(err)
			}
		}
		return nil
	})
	if err != nil {
		return "", ErrHashingBuildContext.Wrap(err)
	}

	logrus.Debug("Generated image hash: ", fmt.Sprintf("%x", hasher.Sum(nil)))

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
