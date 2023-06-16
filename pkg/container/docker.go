// Package container provides utilities for interacting with containers.
package container

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// BuilderFactory is responsible for creating new instances of buildah.Builder
type BuilderFactory struct {
	imageNameFrom          string
	imageNameTo            string
	cli                    *client.Client
	dockerFileInstructions []string
}

// NewBuilderFactory creates a new instance of BuilderFactory.
func NewBuilderFactory(imageName string) (*BuilderFactory, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return &BuilderFactory{
		imageNameFrom:          imageName,
		cli:                    cli,
		dockerFileInstructions: []string{"FROM " + imageName},
	}, nil
}

// ImageNameFrom returns the name of the image from which the builder is created.
func (f *BuilderFactory) ImageNameFrom() string {
	return f.imageNameFrom
}

// ExecuteCmdInBuilder runs the provided command in the context of the given builder.
// It returns the command's output or any error encountered.
func (f *BuilderFactory) ExecuteCmdInBuilder(command []string) (string, error) {
	f.dockerFileInstructions = append(f.dockerFileInstructions, "RUN "+strings.Join(command, " "))
	// FIXME: does not return expected output
	return "", nil
}

// AddToBuilder adds a file from the source path to the destination path in the image, with the specified ownership.
func (f *BuilderFactory) AddToBuilder(srcPath, destPath, chown string) error {
	f.dockerFileInstructions = append(f.dockerFileInstructions, "ADD --chown="+chown+" "+srcPath+" "+destPath)
	return nil
}

// ReadFileFromBuilder reads a file from the given builder's mount point.
// It returns the file's content or any error encountered.
func (f *BuilderFactory) ReadFileFromBuilder(filePath string) ([]byte, error) {
	if f.imageNameTo == "" {
		return nil, fmt.Errorf("no image name provided, push before reading")
	}
	containerConfig := &container.Config{
		Image: f.imageNameTo,
		Cmd:   []string{"tail", "-f", "/dev/null"}, // This keeps the container running
	}
	resp, err := f.cli.ContainerCreate(
		context.Background(),
		containerConfig,
		nil,
		nil,
		nil,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	defer func() {
		// Stop the container
		timeout := int(time.Duration(10) * time.Second)
		stopOptions := container.StopOptions{
			Timeout: &timeout,
		}

		if err := f.cli.ContainerStop(context.Background(), resp.ID, stopOptions); err != nil {
			logrus.Warnf("failed to stop container: %v", err)
		}

		// Remove the container
		if err := f.cli.ContainerRemove(context.Background(), resp.ID, types.ContainerRemoveOptions{}); err != nil {
			logrus.Warnf("failed to remove container: %v", err)
		}
	}()

	if err := f.cli.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Now you can copy the file
	reader, _, err := f.cli.CopyFromContainer(context.Background(), resp.ID, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file from container: %w", err)
	}
	defer reader.Close()

	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read from tar: %w", err)
		}

		if header.Typeflag == tar.TypeReg { // if it's a file then extract it
			data, err := io.ReadAll(tarReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read file from tar: %w", err)
			}
			return data, nil
		}
	}

	return nil, fmt.Errorf("file not found in tar")
}

// SetEnvVar sets the value of an environment variable in the builder.
func (f *BuilderFactory) SetEnvVar(name, value string) error {
	f.dockerFileInstructions = append(f.dockerFileInstructions, "ENV "+name+"="+value)
	return nil
}

// SetUser sets the user in the builder.
func (f *BuilderFactory) SetUser(user string) error {
	f.dockerFileInstructions = append(f.dockerFileInstructions, "USER "+user)
	return nil
}

// Changed returns true if the builder has been modified, false otherwise.
func (f *BuilderFactory) Changed() bool {
	return len(f.dockerFileInstructions) > 1
}

// PushBuilderImage pushes the image from the given builder to a registry.
// The image is identified by the provided name.
func (f *BuilderFactory) PushBuilderImage(imageName string) error {

	if !f.Changed() {
		logrus.Debugf("No changes made to image %s, skipping push", f.imageNameFrom)
		return nil
	}

	// Generate a UUID
	uuid, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("failed to generate UUID: %w", err)
	}

	f.imageNameTo = imageName

	// Create a Dockerfile with a unique name
	dockerFileName := fmt.Sprintf("Dockerfile_%s", uuid.String())
	dockerFile := strings.Join(f.dockerFileInstructions, "\n")
	err = os.WriteFile(dockerFileName, []byte(dockerFile), 0644)
	if err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	// Check if there is an existing builder instance
	cmd := exec.Command("docker", "buildx", "ls")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list buildx builders: %w", err)
	}

	// If no builder instance exists, create a new one
	if !strings.Contains(string(output), "default") {
		cmd = exec.Command("docker", "buildx", "create", "--use")
		err = runCommand(cmd)
		if err != nil {
			return fmt.Errorf("failed to create buildx builder: %w", err)
		}
	}

	// Build the Docker image using buildx
	cmd = exec.Command("docker", "buildx", "build", "--load", "--platform", "linux/amd64", "-t", imageName, ".", "-f", dockerFileName)
	err = runCommand(cmd)
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}

	// Push the Docker image to the registry
	cmd = exec.Command("docker", "push", imageName)
	err = runCommand(cmd)
	if err != nil {
		return fmt.Errorf("failed to push image: %w", err)
	}

	// Remove the Dockerfile
	err = os.Remove(dockerFileName)
	if err != nil {
		return fmt.Errorf("failed to remove Dockerfile: %w", err)
	}

	return nil
}

func runCommand(cmd *exec.Cmd) error {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("command failed: %s\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	return nil
}

func addDirToTar(tw *tar.Writer, dirPath string) error {
	dir, err := os.Open(dirPath)
	if err != nil {
		return err
	}
	defer dir.Close()

	files, err := dir.Readdir(0)
	if err != nil {
		return err
	}

	for _, fileInfo := range files {
		if fileInfo.IsDir() {
			addDirToTar(tw, dirPath+"/"+fileInfo.Name())
		} else {
			file, err := os.Open(dirPath + "/" + fileInfo.Name())
			if err != nil {
				return err
			}
			defer file.Close()

			h := &tar.Header{
				Name: dirPath + "/" + fileInfo.Name(),
				Size: fileInfo.Size(),
				Mode: int64(fileInfo.Mode()),
			}
			err = tw.WriteHeader(h)
			if err != nil {
				return err
			}

			_, err = io.Copy(tw, file)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
