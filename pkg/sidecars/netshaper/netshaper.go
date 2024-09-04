package netshaper

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/bittwister/sdk"

	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/system"
)

const (
	DefaultPort             = 9009
	DefaultImage            = "ghcr.io/celestiaorg/bittwister:4187779"
	DefaultNetworkInterface = "eth0"

	waitToStartInterval = 50 * time.Millisecond
	instanceName        = "bit-twister"
	envServeAddr        = "SERVE_ADDR"
	capabilityNetAdmin  = "NET_ADMIN"
)

type NetShaper struct {
	instance         *instance.Instance
	port             int
	image            string
	networkInterface string
	client           *sdk.Client
}

var _ instance.SidecarManager = (*NetShaper)(nil)

func New() *NetShaper {
	return &NetShaper{
		port:             DefaultPort,
		image:            DefaultImage,
		networkInterface: DefaultNetworkInterface,
	}
}

// Initialize initializes the BitTwister sidecar
// and it is called once the instance.AddSidecar is called
func (bt *NetShaper) Initialize(ctx context.Context, sysDeps system.SystemDependencies) error {
	var err error
	bt.instance, err = instance.New(instanceName, sysDeps)
	if err != nil {
		return ErrCreatingBitTwisterInstance.Wrap(err)
	}
	bt.instance.Sidecars().SetIsSidecar(true)

	if err := bt.instance.Build().SetImage(ctx, bt.image); err != nil {
		return ErrSettingBitTwisterImage.Wrap(err)
	}

	if err := bt.instance.Network().AddPortTCP(bt.port); err != nil {
		return ErrAddingBitTwisterPort.Wrap(err)
	}

	if err := bt.instance.Build().Commit(ctx); err != nil {
		return ErrCommittingBitTwisterInstance.Wrap(err)
	}

	if err := bt.instance.Build().SetEnvironmentVariable(
		envServeAddr, fmt.Sprintf("0.0.0.0:%d", bt.port),
	); err != nil {
		return ErrSettingBitTwisterEnv.Wrap(err)
	}

	if err := bt.instance.Security().SetPrivileged(true); err != nil {
		return ErrSettingBitTwisterPrivileged.Wrap(err)
	}

	if err := bt.instance.Security().AddKubernetesCapability(capabilityNetAdmin); err != nil {
		return ErrAddingBitTwisterCapability.Wrap(err)
	}

	return nil
}

// PreStart is called before the instance is started
// It is used to prepare the sidecar for the instance to start
func (bt *NetShaper) PreStart(ctx context.Context) error {
	if bt.instance == nil {
		return ErrBitTwisterNotInitialized
	}

	btURL, err := bt.instance.Network().AddHost(ctx, bt.port)
	if err != nil {
		return err
	}

	bt.setNewClientByURL(btURL)
	return nil
}

func (bt *NetShaper) Instance() *instance.Instance {
	return bt.instance
}

func (bt *NetShaper) CloneWithSuffix(suffix string) instance.SidecarManager {
	return &NetShaper{
		instance:         bt.instance.CloneWithSuffix(suffix),
		port:             bt.port,
		image:            bt.image,
		networkInterface: bt.networkInterface,
	}
}
