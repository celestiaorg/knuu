package bittwister

import (
	"context"
	"time"

	"github.com/celestiaorg/bittwister/sdk"
)

func (bt *BitTwister) setNewClientByURL(url string) {
	bt.client = sdk.NewClient(url)
	bt.instance.Logger.Debugf("BitTwister address '%s'", url)
}

func (bt *BitTwister) SetPort(port int) {
	bt.port = port
}

func (bt *BitTwister) SetImage(image string) {
	bt.image = image
}

func (bt *BitTwister) SetNetworkInterface(netIf string) {
	bt.networkInterface = netIf
}

// SetBandwidthLimit sets the bandwidth limit of the instance
// bandwidth limit in bps (e.g. 1000 for 1Kbps)
// Currently, only one of bandwidth, jitter, latency or packet loss can be set
// This function can only be called in the state 'Commited'
func (bt *BitTwister) SetBandwidthLimit(limit int64) error {
	if bt.client == nil {
		return ErrBitTwisterNotInitialized
	}

	// We first need to stop it, otherwise we get an error
	if err := bt.client.BandwidthStop(); err != nil {
		if !sdk.IsErrorServiceNotInitialized(err) &&
			!sdk.IsErrorServiceNotReady(err) &&
			!sdk.IsErrorServiceNotStarted(err) {
			return ErrStoppingBandwidthLimit.WithParams(bt.instance.Name()).Wrap(err)
		}
	}

	return bt.client.BandwidthStart(sdk.BandwidthStartRequest{
		NetworkInterfaceName: bt.networkInterface,
		Limit:                limit,
	})
}

// SetLatency sets the latency of the instance
// latency in ms (e.g. 1000 for 1s)
// jitter in ms (e.g. 1000 for 1s)
func (bt *BitTwister) SetLatencyAndJitter(latency, jitter int64) error {
	if bt.client == nil {
		return ErrBitTwisterNotInitialized
	}

	// We first need to stop it, otherwise we get an error
	if err := bt.client.LatencyStop(); err != nil {
		if !sdk.IsErrorServiceNotInitialized(err) &&
			!sdk.IsErrorServiceNotReady(err) &&
			!sdk.IsErrorServiceNotStarted(err) {
			return ErrStoppingLatencyJitter.WithParams(bt.instance.Name()).Wrap(err)
		}
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
func (bt *BitTwister) SetPacketLoss(packetLoss int32) error {
	if bt.client == nil {
		return ErrBitTwisterNotInitialized
	}
	// We first need to stop it, otherwise we get an error
	if err := bt.client.PacketlossStop(); err != nil {
		if !sdk.IsErrorServiceNotInitialized(err) &&
			!sdk.IsErrorServiceNotReady(err) &&
			!sdk.IsErrorServiceNotStarted(err) {
			return ErrStoppingPacketLoss.WithParams(bt.instance.Name()).Wrap(err)
		}
	}

	return bt.client.PacketlossStart(sdk.PacketLossStartRequest{
		NetworkInterfaceName: bt.networkInterface,
		PacketLossRate:       packetLoss,
	})
}

func (bt *BitTwister) WaitForStart(ctx context.Context) error {
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

func (bt *BitTwister) AllServicesStatus() ([]sdk.ServiceStatus, error) {
	if bt.client == nil {
		return nil, ErrBitTwisterNotInitialized
	}
	return bt.client.AllServicesStatus()
}
