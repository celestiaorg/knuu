package container

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"io"
	"os"
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

// ExecuteCmdInBuilder runs the provided command in the context of the given builder.
// It returns the command's output or any error encountered.
func (f *BuilderFactory) ExecuteCmdInBuilder(command []string) (string, error) {
	f.dockerFileInstructions = append(f.dockerFileInstructions, "RUN "+strings.Join(command, " "))
	// FIXME: does not return expected output
	return "", nil
}

// AddFileToBuilder adds a file from the source path to the destination path in the image, with the specified ownership.
func (f *BuilderFactory) AddFileToBuilder(srcPath, destPath, chown string) error {
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

// PushBuilderImage pushes the image from the given builder to a registry.
// The image is identified by the provided name.
func (f *BuilderFactory) PushBuilderImage(imageName string) error {

	f.imageNameTo = imageName

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	dockerFile := strings.Join(f.dockerFileInstructions, "\n")
	dockerFileReader := strings.NewReader(dockerFile)
	tw.WriteHeader(&tar.Header{
		Name: "Dockerfile",
		Size: int64(dockerFileReader.Len()),
	})
	io.Copy(tw, dockerFileReader)

	addDirToTar(tw, "./")

	buildOptions := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{imageName},
		Platform:   "linux/amd64",
	}

	buildResponse, err := f.cli.ImageBuild(context.Background(), buf, buildOptions)
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}
	defer buildResponse.Body.Close()

	// Create a Scanner to read the build output line by line
	scanner := bufio.NewScanner(buildResponse.Body)

	type ErrorMessage struct {
		Error string
	}
	var errorMessage ErrorMessage

	for scanner.Scan() {
		// Each line is a JSON object, so we can unmarshal it
		var line map[string]interface{}
		err := json.Unmarshal(scanner.Bytes(), &line)
		if err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		// If there's an error, return it
		if err, ok := line["error"].(string); ok {
			errorMessage.Error = err
			return fmt.Errorf("failed to build image: %v", errorMessage)
		}
	}

	// Prepare an empty AuthConfig
	authConfig := registry.AuthConfig{}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		panic(err)
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	pushOptions := types.ImagePushOptions{
		RegistryAuth: authStr,
	}

	out, err := f.cli.ImagePush(context.Background(), imageName, pushOptions)
	if err != nil {
		return fmt.Errorf("failed to push image: %w", err)
	}

	defer out.Close()

	//type ErrorMessage struct {
	//	Error string
	//}
	//var errorMessage ErrorMessage
	buffIOReader := bufio.NewReader(out)

	for {
		streamBytes, err := buffIOReader.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		json.Unmarshal(streamBytes, &errorMessage)
		if errorMessage.Error != "" {
			return fmt.Errorf("failed to push image: %s", errorMessage.Error)
		}
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
