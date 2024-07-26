package tshark

import (
	"github.com/celestiaorg/knuu/pkg/errors"
)

type Error = errors.Error

var (
	ErrCreatingTsharkCollectorInstance      = errors.New("CreatingTsharkCollectorInstance", "error creating tshark collector instance")
	ErrSettingTsharkCollectorImage          = errors.New("SettingTsharkCollectorImage", "error setting image for tshark collector")
	ErrSettingTsharkCollectorMemory         = errors.New("SettingTsharkCollectorMemory", "error setting memory for tshark collector")
	ErrAddingTsharkCollectorVolume          = errors.New("AddingTsharkCollectorVolume", "error adding volume for tshark collector")
	ErrSettingTsharkCollectorEnv            = errors.New("SettingTsharkCollectorEnv", "error setting environment variables for tshark collector")
	ErrAddingTsharkCollectorCapability      = errors.New("AddingTsharkCollectorCapability", "error adding capability for tshark collector")
	ErrCommittingTsharkCollectorInstance    = errors.New("CommittingTsharkCollectorInstance", "error committing tshark collector instance")
	ErrTsharkCollectorNotInitialized        = errors.New("TsharkCollectorNotInitialized", "tshark collector not initialized")
	ErrTsharkCollectorInvalidVolumeSize     = errors.New("TsharkCollectorInvalidVolumeSize", "tshark collector invalid volume size `%s`")
	ErrTsharkCollectorS3RegionOrBucketEmpty = errors.New("TsharkCollectorS3RegionOrBucketEmpty", "tshark collector s3 region or bucket empty")
	ErrSettingTsharkCollectorCPU            = errors.New("SettingTsharkCollectorCPU", "error setting cpu for tshark collector")
)
