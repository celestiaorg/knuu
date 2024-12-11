package registry

import (
	"encoding/base64"
	"fmt"
)

const (
	dockerHubRegistryAddress = "docker.io"
	dockerHubRegistryPushURL = "https://index.docker.io/v1/"
)

type DockerHub struct {
	username   string
	password   string
	repository string
}

var _ Registry = &DockerHub{}

func NewDockerHub(username, password, repository string) (*DockerHub, error) {
	if username == "" || password == "" {
		return nil, ErrUsernamePasswordRequired
	}

	if repository == "" {
		repository = username
	}

	return &DockerHub{
		username:   username,
		password:   password,
		repository: repository,
	}, nil
}

func (s *DockerHub) Address() string {
	return dockerHubRegistryAddress
}

func (s *DockerHub) GenerateConfig() ([]byte, error) {
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", s.username, s.password)))
	return []byte(fmt.Sprintf(`{
        "auths": {
            "%s": {
                "auth": "%s"
            }
        }
    }`, dockerHubRegistryPushURL, auth)), nil

}

func (s *DockerHub) ResolvedImage(imageName, tag string) *ResolvedImage {
	return &ResolvedImage{
		Prefix: fmt.Sprintf("%s/%s", s.Address(), s.repository),
		Name:   imageName,
		Tag:    tag,
	}
}
