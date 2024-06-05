package docker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/celestiaorg/knuu/pkg/builder"
)

type Docker struct {
	K8sClientset kubernetes.Interface
	K8sNamespace string
}

var _ builder.Builder = &Docker{}

func (d *Docker) Build(_ context.Context, b *builder.BuilderOptions) (logs string, err error) {
	if builder.IsGitContext(b.BuildContext) {
		return "", ErrGitContextNotSupported
	}

	// Check if there is an existing builder instance
	cmd := exec.Command("docker", "buildx", "ls")
	output, err := cmd.Output()
	logrus.Debugf("docker buildx ls: %s", output)
	if err != nil {
		return "", ErrFailedToListBuildxBuilders.Wrap(err)
	}

	// If no builder instance exists, create a new one
	if !strings.Contains(string(output), "default") {
		cmd = exec.Command("docker", "buildx", "create", "--use")
		if _, err := runCommand(cmd); err != nil {
			return "", ErrFailedToCreateBuilder.Wrap(err)
		}
		logrus.Debug("created new docker builder instance")
	}

	logrus.Debug("building docker image: ", b.Destination)

	buildContext := builder.GetDirFromBuildContext(b.BuildContext)

	// Since in docker the image name and destination must be the same, we just use the destination as the image name
	cmd = exec.Command("docker", "buildx", "build", "--load", "--platform", "linux/amd64", "-t", b.Destination, buildContext)
	cmdLogs, err := runCommand(cmd)
	if err != nil {
		return "", ErrFailedToBuildImage.Wrap(err)
	}
	logs += cmdLogs + "\n"
	logrus.Debug("built docker image: ", b.Destination)
	logrus.Debug("logs: ", cmdLogs)

	cmd = exec.Command("docker", "push", b.Destination)
	cmdLogs, err = runCommand(cmd)
	if err != nil {
		return "", ErrFailedToPushImage.Wrap(err)
	}
	logs += cmdLogs + "\n"
	logrus.Debug("pushed docker image: ", b.Destination)
	logrus.Debug("logs: ", cmdLogs)

	if err := os.RemoveAll(b.BuildContext); err != nil {
		return "", ErrFailedToRemoveContextDir.Wrap(err)
	}

	return logs, nil
}

func runCommand(cmd *exec.Cmd) (logs string, err error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", ErrRunCommandFailed.Wrap(fmt.Errorf("%w\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String()))
	}
	return stdout.String() + stderr.String(), nil
}
