package tshark

// validateConfig checks the configuration fields for proper formatting
func (t *Tshark) validateConfig() error {
	if t.VolumeSize.IsZero() {
		return ErrTsharkCollectorInvalidVolumeSize.
			WithParams(t.VolumeSize.String())
	}
	if t.S3Region == "" || t.S3Bucket == "" {
		return ErrTsharkCollectorS3RegionOrBucketEmpty.
			WithParams(t.S3Region, t.S3Bucket)
	}

	return nil
}
