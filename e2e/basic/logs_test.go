package basic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/system"
)

const expectedLogMsg = "Hello World"

type sidecarLogsTest struct {
	instance *instance.Instance
}

var _ instance.SidecarManager = (*sidecarLogsTest)(nil)

func (s *Suite) TestLogs() {
	const namePrefix = "logs"
	ctx := context.Background()

	// Create a new instance
	target, err := s.Knuu.NewInstance(namePrefix + "-target")
	s.Require().NoError(err)

	// Set the image and start command to generate logs
	s.Require().NoError(target.Build().SetImage(ctx, alpineImage))
	s.Require().NoError(target.Build().SetStartCommand("sh", "-c", fmt.Sprintf("while true; do echo '%s'; sleep 1; done", expectedLogMsg)))
	s.Require().NoError(target.Build().Commit(ctx))
	s.Require().NoError(target.Execution().Start(ctx))

	// Wait for a short duration to allow log generation
	time.Sleep(5 * time.Second)

	logStream, err := target.Monitoring().Logs(ctx)
	s.Require().NoError(err)
	defer logStream.Close()

	logs, err := io.ReadAll(logStream)
	s.Require().NoError(err)

	logOutput := string(logs)
	s.Contains(logOutput, expectedLogMsg)
}

func (s *Suite) TestLogsWithSidecar() {
	const namePrefix = "logs-sidecar"
	ctx := context.Background()

	// Create a new instance
	target, err := s.Knuu.NewInstance(namePrefix + "-target")
	s.Require().NoError(err)

	sidecar := &sidecarLogsTest{}

	s.Require().NoError(target.Build().SetImage(ctx, alpineImage))
	s.Require().NoError(target.Build().SetStartCommand("sh", "-c", "sleep infinity"))
	s.Require().NoError(target.Build().Commit(ctx))
	s.Require().NoError(target.Sidecars().Add(ctx, sidecar))
	s.Require().NoError(target.Execution().Start(ctx))

	// Wait for a short duration to allow log generation
	time.Sleep(5 * time.Second)

	logStream, err := sidecar.Instance().Monitoring().Logs(ctx)
	s.Require().NoError(err)
	defer logStream.Close()

	logs, err := io.ReadAll(logStream)
	s.Require().NoError(err)

	logOutput := string(logs)
	s.Contains(logOutput, expectedLogMsg)
}

func (sl *sidecarLogsTest) Initialize(ctx context.Context, namePrefix string, sysDeps *system.SystemDependencies) error {
	var err error
	sl.instance, err = instance.New(namePrefix+"-sidecar-logs", sysDeps)
	if err != nil {
		return err
	}
	sl.instance.Sidecars().SetIsSidecar(true)

	if err := sl.instance.Build().SetImage(ctx, alpineImage); err != nil {
		return err
	}

	err = sl.instance.Build().SetStartCommand("sh", "-c", fmt.Sprintf("while true; do echo '%s'; sleep 1; done", expectedLogMsg))
	if err != nil {
		return err
	}

	if err := sl.instance.Build().Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (sl *sidecarLogsTest) PreStart(ctx context.Context) error {
	if sl.instance == nil {
		return errors.New("instance not initialized")
	}
	return nil
}

func (sl *sidecarLogsTest) Instance() *instance.Instance {
	return sl.instance
}

func (sl *sidecarLogsTest) Clone(namePrefix string) (instance.SidecarManager, error) {
	clone, err := sl.instance.CloneWithName(namePrefix + "-" + sl.instance.Name())
	if err != nil {
		return nil, err
	}
	return &sidecarLogsTest{
		instance: clone,
	}, nil
}
