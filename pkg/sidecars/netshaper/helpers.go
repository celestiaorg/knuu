package netshaper

import (
	"context"
	"time"

	"github.com/celestiaorg/bittwister/api/v1"
	"github.com/celestiaorg/bittwister/sdk"
)

func (bt *NetShaper) setNewClientByURL(url string) {
	bt.client = sdk.NewClient(url)
	bt.instance.Logger.Debugf("NetShaper (BitTwister) address '%s'", url)
}

func (bt *NetShaper) SetPort(port int) {
	bt.port = port
}

func (bt *NetShaper) SetImage(image string) {
	bt.image = image
}

func (bt *NetShaper) SetNetworkInterface(netIf string) {
	bt.networkInterface = netIf
}

// SetBandwidthLimit sets the bandwidth limit of the instance
// bandwidth limit in bps (e.g. 1000 for 1Kbps)
// Currently, only one of bandwidth, jitter, latency or packet loss can be set
// This function can only be called in the state 'Commited'
func (bt *NetShaper) SetBandwidthLimit(limit int64) error {
	if bt.client == nil {
		return ErrBitTwisterNotInitialized
	}

	err := bt.stopIfRunning(bt.client.BandwidthStatus, bt.client.BandwidthStop)
	if err != nil {
		return err
	}

	return bt.client.BandwidthStart(sdk.BandwidthStartRequest{
		NetworkInterfaceName: bt.networkInterface,
		Limit:                limit,
	})
}

// SetLatency sets the latency of the instance
// latency in ms (e.g. 1000 for 1s)
// jitter in ms (e.g. 1000 for 1s)
func (bt *NetShaper) SetLatencyAndJitter(latency, jitter int64) error {
	if bt.client == nil {
		return ErrBitTwisterNotInitialized
	}

	err := bt.stopIfRunning(bt.client.LatencyStatus, bt.client.LatencyStop)
	if err != nil {
		return err
	}

	return bt.client.LatencyStart(sdk.LatencyStartRequest{
		NetworkInterfaceName: bt.networkInterface,
		Latency:              latency,
		Jitter:               jitter,
	})
}

// SetPacketLoss sets the packet loss of the instance
// packet loss in percent (e.g. 10 for 10%)
// Currently, only one of bandwidth, jitter, latency or packet loss can be set
func (bt *NetShaper) SetPacketLoss(packetLoss int32) error {
	if bt.client == nil {
		return ErrBitTwisterNotInitialized
	}

	err := bt.stopIfRunning(bt.client.PacketlossStatus, bt.client.PacketlossStop)
	if err != nil {
		return err
	}

	return bt.client.PacketlossStart(sdk.PacketLossStartRequest{
		NetworkInterfaceName: bt.networkInterface,
		PacketLossRate:       packetLoss,
	})
}

func (bt *NetShaper) WaitForStart(ctx context.Context) error {
	if bt.client == nil {
		return ErrBitTwisterNotInitialized
	}

	ticker := time.NewTicker(waitToStartInterval)
	defer ticker.Stop()
	for {
		out, err := bt.client.AllServicesStatus()
		if err == nil && len(out) > 0 && len(out[0].Name) > 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (bt *NetShaper) AllServicesStatus() ([]sdk.ServiceStatus, error) {
	if bt.client == nil {
		return nil, ErrBitTwisterNotInitialized
	}
	return bt.client.AllServicesStatus()
}

func (bt *NetShaper) stopIfRunning(
	statusFunc func() (*api.MetaMessage, error),
	stopFunc func() error,
) error {
	// if bt.instance == nil, then the service is not running
	// so we don't need to stop it
	if bt.instance == nil {
		return nil
	}

	status, err := statusFunc()
	if err != nil {
		return ErrGettingServiceStatus.WithParams(bt.instance.Name()).Wrap(err)
	}

	if status.Slug != api.SlugServiceReady {
		return nil
	}

	err = stopFunc()
	if err == nil {
		return nil
	}
	if !sdk.IsErrorServiceNotInitialized(err) &&
		!sdk.IsErrorServiceNotReady(err) &&
		!sdk.IsErrorServiceNotStarted(err) {
		return ErrStoppingService.WithParams(bt.instance.Name()).Wrap(err)
	}
	return err
}
