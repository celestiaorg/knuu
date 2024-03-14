package knuu

import (
	"fmt"

	"github.com/celestiaorg/bittwister/sdk"
	"github.com/sirupsen/logrus"
)

const (
	btDefaultPort             = 9009
	btDefaultImage            = "ghcr.io/celestiaorg/bittwister:4187779"
	btDefaultNetworkInterface = "eth0"
)

type btConfig struct {
	port             int
	image            string
	networkInterface string
	client           *sdk.Client
	enabled          bool // if true, BitTwister is enabled and will be deployed as a sidecar
}

func getBitTwisterDefaultConfig() *btConfig {
	return &btConfig{
		port:             btDefaultPort,
		image:            btDefaultImage,
		networkInterface: btDefaultNetworkInterface,
	}
}

func (c *btConfig) SetPort(port int) {
	c.port = port
}

func (c *btConfig) SetImage(image string) {
	c.image = image
}

func (c *btConfig) SetNetworkInterface(networkInterface string) {
	c.networkInterface = networkInterface
}

func (c *btConfig) SetClient(client *sdk.Client) {
	c.client = client
}

func (c *btConfig) SetNewClientByIPAddr(ip string) {
	btAddress := fmt.Sprintf("%s:%d", ip, c.port)
	c.client = sdk.NewClient(btAddress)
	logrus.Debugf("BitTwister address '%s'", btAddress)
}

func (c *btConfig) Port() int {
	return c.port
}

func (c *btConfig) Image() string {
	return c.image
}

func (c *btConfig) NetworkInterface() string {
	return c.networkInterface
}

func (c *btConfig) Client() *sdk.Client {
	return c.client
}

func (c *btConfig) Enabled() bool {
	return c.enabled
}

func (c *btConfig) enable() {
	c.enabled = true
}

func (c *btConfig) disable() {
	c.enabled = false
}
