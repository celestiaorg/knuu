package registry

import (
	"encoding/base64"
	"fmt"
)

const (
	scalewayRegistryAddress = "rg.%s.scw.cloud"
)

type Scaleway struct {
	region    string
	namespace string
	username  string
	password  string
}

var _ Registry = &Scaleway{}

func NewScaleway(region, namespace, username, password string) (*Scaleway, error) {
	if region == "" || namespace == "" || username == "" || password == "" {
		return nil, ErrRegionNamespaceUsernamePasswordRequired
	}

	return &Scaleway{
		region:    region,
		namespace: namespace,
		username:  username,
		password:  password,
	}, nil
}

func (s *Scaleway) Address() string {
	return fmt.Sprintf(scalewayRegistryAddress, s.region)
}

func (s *Scaleway) GenerateConfig() ([]byte, error) {
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", s.username, s.password)))

	return []byte(fmt.Sprintf(`{
        "auths": {
            "https://%s": {
                "auth": "%s"
            }
        }
    }`, s.Address(), auth)), nil

}

func (s *Scaleway) ResolvedImage(imageName, tag string) *ResolvedImage {
	return &ResolvedImage{
		Prefix: fmt.Sprintf("%s/%s", s.Address(), s.namespace),
		Name:   imageName,
		Tag:    tag,
	}
}
