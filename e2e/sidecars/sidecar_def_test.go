package sidecars

import (
	"context"
	"errors"

	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/system"
)

type testSidecar struct {
	instance     *instance.Instance
	StartCommand []string
}

var _ instance.SidecarManager = (*testSidecar)(nil)

func (s *testSidecar) Initialize(ctx context.Context, namePrefix string, sysDeps *system.SystemDependencies) error {
	var err error
	s.instance, err = instance.New(namePrefix+"-sidecar-logs", sysDeps)
	if err != nil {
		return err
	}
	s.instance.Sidecars().SetIsSidecar(true)

	if err := s.instance.Build().SetImage(ctx, alpineImage); err != nil {
		return err
	}

	err = s.instance.Build().SetStartCommand(s.StartCommand...)
	if err != nil {
		return err
	}

	if err := s.instance.Build().Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *testSidecar) PreStart(ctx context.Context) error {
	if s.instance == nil {
		return errors.New("instance not initialized")
	}
	if len(s.StartCommand) == 0 {
		return errors.New("start command not configured")
	}
	return nil
}

func (s *testSidecar) Instance() *instance.Instance {
	return s.instance
}

func (s *testSidecar) Clone(namePrefix string) (instance.SidecarManager, error) {
	clone, err := s.instance.CloneWithName(namePrefix + "-" + s.instance.Name())
	if err != nil {
		return nil, err
	}
	return &testSidecar{
		instance:     clone,
		StartCommand: append([]string{}, s.StartCommand...),
	}, nil
}
