package instance

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/celestiaorg/bittwister/sdk"
)

const (
	btDefaultPort             = 9009
	btDefaultImage            = "ghcr.io/celestiaorg/bittwister:4187779"
	btDefaultNetworkInterface = "eth0"
	btWaitToStartInterval     = 50 * time.Millisecond
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

func (c *btConfig) SetNewClientByURL(url string) {
	c.client = sdk.NewClient(url)
	logrus.Debugf("BitTwister address '%s'", url)
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

func (c *btConfig) Started() bool {
	_, err := c.client.AllServicesStatus()
	return err == nil
}

func (c *btConfig) WaitForStart(ctx context.Context) error {
	ticker := time.NewTicker(btWaitToStartInterval)
	defer ticker.Stop()
	for range ticker.C {
		if c.Started() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ErrBitTwisterFailedToStart
		default:
		}
	}
	return nil
}
