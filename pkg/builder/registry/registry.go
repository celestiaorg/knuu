package registry

import "fmt"

type Registry interface {
	Address() string
	GenerateConfig() ([]byte, error) // Generate a json config for the registry
	ResolvedImage(imageName, tag string) *ResolvedImage
}

type ResolvedImage struct {
	Prefix string // could be registry address + repository/namespace...
	Name   string
	Tag    string
}

func (r *ResolvedImage) ToString() string {
	if r.Tag == "" {
		// If no tag is provided, we use the image name only as for the kaniko cache tag is not used
		return fmt.Sprintf("%s/%s", r.Prefix, r.Name)
	}
	return fmt.Sprintf("%s/%s:%s", r.Prefix, r.Name, r.Tag)
}
