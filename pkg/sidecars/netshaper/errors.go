package netshaper

import (
	"github.com/celestiaorg/knuu/pkg/errors"
)

type Error = errors.Error

var (
	ErrBitTwisterFailedToStart                   = errors.New("BitTwisterFailedToStart", "BitTwister failed to start")
	ErrSettingBitTwisterImage                    = errors.New("SettingBitTwisterImage", "error setting image for bit-twister instance")
	ErrAddingBitTwisterPort                      = errors.New("AddingBitTwisterPort", "error adding BitTwister port")
	ErrCommittingBitTwisterInstance              = errors.New("CommittingBitTwisterInstance", "error committing bit-twister instance")
	ErrSettingBitTwisterEnv                      = errors.New("SettingBitTwisterEnv", "error setting environment variable for bit-twister instance")
	ErrCreatingBitTwisterInstance                = errors.New("CreatingBitTwisterInstance", "error creating bit-twister instance '%s'")
	ErrSettingBitTwisterPrivileged               = errors.New("SettingBitTwisterPrivileged", "error setting privileged for bit-twister instance '%s'")
	ErrAddingBitTwisterCapability                = errors.New("AddingBitTwisterCapability", "error adding capability for bit-twister instance '%s'")
	ErrAddingBitTwisterSidecar                   = errors.New("AddingBitTwisterSidecar", "error adding bit-twister sidecar to instance '%s'")
	ErrEnablingBitTwister                        = errors.New("EnablingBitTwister", "enabling BitTwister is not allowed in state 'Started'")
	ErrSettingBandwidthLimitNotAllowedBitTwister = errors.New("SettingBandwidthLimitNotAllowedBitTwister", "setting bandwidth limit is only allowed if BitTwister is enabled")
	ErrSettingLatencyJitterNotAllowedBitTwister  = errors.New("SettingLatencyJitterNotAllowedBitTwister", "setting latency/jitter is only allowed if BitTwister is enabled")
	ErrSettingPacketLossNotAllowedBitTwister     = errors.New("SettingPacketLossNotAllowedBitTwister", "setting packetloss is only allowed if BitTwister is enabled")
	ErrAddingToProxy                             = errors.New("AddingToProxy", "error adding '%s' to proxy '%s'")
	ErrStoppingBandwidthLimit                    = errors.New("StoppingBandwidthLimit", "error stopping bandwidth limit for bit-twister instance '%s'")
	ErrBitTwisterNotInitialized                  = errors.New("BitTwisterNotInitialized", "bit-twister instance '%s' not initialized")
	ErrStoppingLatencyJitter                     = errors.New("StoppingLatencyJitter", "error stopping latency/jitter for bit-twister instance '%s'")
	ErrStoppingPacketLoss                        = errors.New("StoppingPacketLoss", "error stopping packet loss for bit-twister instance '%s'")
	ErrGettingServiceStatus                      = errors.New("GettingServiceStatus", "error getting service status for net-shaper (bit-twister) instance '%s'")
	ErrStoppingService                           = errors.New("StoppingService", "error stopping service for net-shaper (bit-twister) instance '%s'")
)
