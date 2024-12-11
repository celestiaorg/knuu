package registry

const (
	defaultRegistryAddress = "ttl.sh"
)

type Default struct {
	address string
}

var _ Registry = &Default{}

func NewDefault() *Default {
	return &Default{
		address: defaultRegistryAddress,
	}
}

func (d *Default) Address() string {
	return d.address
}

func (d *Default) GenerateConfig() ([]byte, error) {
	return nil, nil
}

func (d *Default) ResolvedImage(imageName, tag string) *ResolvedImage {
	return &ResolvedImage{
		Prefix: d.Address(),
		Name:   imageName,
		Tag:    tag,
	}
}
