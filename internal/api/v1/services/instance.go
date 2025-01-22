package services

import (
	"context"
	"fmt"

	"github.com/celestiaorg/knuu/pkg/builder"
)

type Instance struct {
	Name         string             `json:"name" binding:"required"`
	Scope        string             `json:"scope" binding:"required"`
	Image        string             `json:"image"`
	GitContext   builder.GitContext `json:"git_context"`
	BuildArgs    []string           `json:"build_args"`
	StartCommand []string           `json:"start_command"`
	Args         []string           `json:"args"`
	Status       string             `json:"status"` // Readonly
	StartNow     bool               `json:"start_now"`
	Env          map[string]string  `json:"env"`
	TCPPorts     []int              `json:"tcp_ports"`
	UDPPorts     []int              `json:"udp_ports"`
	Hostname     string             `json:"hostname"` // Readonly

	// Volumes      []k8s.Volume       `json:"volumes"`
}

func (s *TestService) CreateInstance(ctx context.Context, userID uint, instance *Instance) error {
	if userID == 0 {
		return ErrUserIDRequired
	}

	kn, err := s.Knuu(userID, instance.Scope)
	if err != nil {
		return err
	}

	ins, err := kn.NewInstance(instance.Name)
	if err != nil {
		return err
	}

	buildArgs := []builder.ArgInterface{}
	for _, arg := range instance.BuildArgs {
		buildArgs = append(buildArgs, &builder.BuildArg{Value: arg})
	}

	if instance.Image != "" {
		if err := ins.Build().SetImage(ctx, instance.Image, buildArgs...); err != nil {
			return err
		}
	}

	if len(instance.StartCommand) > 0 {
		if err := ins.Build().SetStartCommand(instance.StartCommand...); err != nil {
			return err
		}
	}

	if len(instance.Args) > 0 {
		if err := ins.Build().SetArgs(instance.Args...); err != nil {
			return err
		}
	}

	for k, v := range instance.Env {
		if err := ins.Build().SetEnvironmentVariable(k, v); err != nil {
			return err
		}
	}

	if instance.GitContext.Repo != "" {
		if err := ins.Build().SetGitRepo(ctx, instance.GitContext, buildArgs...); err != nil {
			return err
		}
	}

	for _, port := range instance.TCPPorts {
		if err := ins.Network().AddPortTCP(port); err != nil {
			return err
		}
	}

	for _, port := range instance.UDPPorts {
		if err := ins.Network().AddPortUDP(port); err != nil {
			return err
		}
	}

	if instance.StartNow {
		return ins.Execution().StartAsync(ctx)
	}
	return nil
}

func (s *TestService) GetInstance(ctx context.Context, userID uint, scope, instanceName string) (*Instance, error) {
	kn, err := s.Knuu(userID, scope)
	if err != nil {
		return nil, err
	}

	_ = kn

	return nil, fmt.Errorf("not implemented")
}
