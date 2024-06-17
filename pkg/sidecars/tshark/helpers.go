package tshark

import (
	"k8s.io/apimachinery/pkg/api/resource"
)

// validateConfig checks the configuration fields for proper formatting
func (t *Tshark) validateConfig() error {
	_, err := resource.ParseQuantity(t.VolumeSize)
	if err != nil {
		return ErrTsharkCollectorInvalidVolumeSize.
			WithParams(t.VolumeSize).Wrap(err)
	}
	if t.S3Region == "" || t.S3Bucket == "" {
		return ErrTsharkCollectorS3RegionOrBucketEmpty.
			WithParams(t.S3Region, t.S3Bucket)
	}

	return nil
}
