// Package registry provides a local registry for building images.
package registry

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/celestiaorg/knuu/pkg/cert"
	"github.com/celestiaorg/knuu/pkg/instance"
)

const (
	TtlShSecretName   = "ttlsh-registry-certs"
	TtlShInstanceName = "ttlsh-registry"

	ttlShDefaultImage       = "mojiz/ttl.sh:dev"
	ttlShDefaultPort        = 5001
	ttlShConfigFile         = "/knuu/config.yml"
	ttlShStoragePath        = "/var/lib/registry"
	ttlShCertFilePath       = "/knuu/certs/cert.pem"
	ttlShKeyFilePath        = "/knuu/certs/key.pem"
	ttlShDefaultStorageSize = "10Gi"
)

type TtlSh struct {
	instance *instance.Instance
	port     int
	cert     []byte
}

type TtlShOptions struct {
	Port        int
	Image       string
	StorageSize resource.Quantity
	LogLevel    string
}

func NewTtlSh(ctx context.Context, ins *instance.Instance, opts TtlShOptions) (*TtlSh, error) {
	if opts.Image == "" {
		opts.Image = ttlShDefaultImage
	}

	if err := ins.Build().SetImage(ctx, opts.Image); err != nil {
		return nil, err
	}

	if err := ins.Build().Commit(ctx); err != nil {
		return nil, err
	}

	if opts.Port == 0 {
		opts.Port = ttlShDefaultPort
	}

	if err := ins.Network().AddPortTCP(opts.Port); err != nil {
		return nil, err
	}

	if opts.StorageSize.IsZero() {
		opts.StorageSize = resource.MustParse(ttlShDefaultStorageSize)
	}

	if err := ins.Storage().AddVolume(ttlShStoragePath, opts.StorageSize); err != nil {
		return nil, err
	}

	if opts.LogLevel == "" {
		opts.LogLevel = ins.Logger.GetLevel().String()
	}

	if err := ins.Storage().AddFileBytes(
		generateTtlShConfig(opts), ttlShConfigFile, "0:0",
	); err != nil {
		return nil, err
	}

	if err := ins.Build().SetStartCommand("registry", "serve", ttlShConfigFile); err != nil {
		return nil, err
	}

	hostname := ins.Network().HostName()
	certPEM, err := setupTLSForTtlSh(ctx, ins, hostname)
	if err != nil {
		return nil, err
	}

	// TLS is enabled, so headless service does not work
	ins.Network().SetHeadless(false)

	if err := ins.Execution().Start(ctx); err != nil {
		return nil, err
	}

	t := &TtlSh{
		instance: ins,
		port:     opts.Port,
		cert:     certPEM,
	}
	ins.Logger.Infof("TTL.sh local registry started at `%s`", t.GetAddress())

	return t, nil
}

func (t *TtlSh) Cert() []byte {
	return t.cert
}

func generateTtlShConfig(opts TtlShOptions) []byte {
	// secret := uuid.New().String() // just a random secret
	return []byte(fmt.Sprintf(
		`version: 0.1
log:
  level: %s

storage:
  filesystem:
    rootdirectory: %s

http:
  addr: 0.0.0.0:%d
  tls:
    certificate: %s
    key: %s
`,
		strings.ToLower(opts.LogLevel),
		ttlShStoragePath,
		opts.Port,
		ttlShCertFilePath,
		ttlShKeyFilePath,
	))
	// 	return []byte(fmt.Sprintf(
	// 		`version: 0.1
	// log:
	//   level: %s

	// storage:
	//   filesystem:
	//     rootdirectory: %s

	// http:
	//
	//	  addr: 0.0.0.0:%d
	//	  secret: "%s"
	//	  tls:
	//	    certificate: %s
	//	    key: %s
	//	  host: "localhost:%d"`,
	//			strings.ToLower(opts.LogLevel),
	//			ttlShStoragePath,
	//			opts.Port,
	//			secret,
	//			ttlShCertFilePath,
	//			ttlShKeyFilePath,
	//			opts.Port,
	//		))
}

func setupTLSForTtlSh(ctx context.Context, ins *instance.Instance, hostname string) (certPEM []byte, err error) {
	certPEM, keyPEM, err := cert.GenerateSelfSignedCert(hostname)
	if err != nil {
		return nil, err
	}

	if err := ins.K8sClient.CreateTLSSecret(ctx, TtlShSecretName, certPEM, keyPEM); err != nil {
		return nil, err
	}

	if err := ins.Storage().AddFileBytes(certPEM, ttlShCertFilePath, "0:0"); err != nil {
		return nil, err
	}

	if err := ins.Storage().AddFileBytes(keyPEM, ttlShKeyFilePath, "0:0"); err != nil {
		return nil, err
	}

	return certPEM, nil
}

func (t *TtlSh) GetAddress() string {
	if t.instance == nil {
		panic("TTL.sh instance is not initialized")
	}
	return fmt.Sprintf("%s:%d", t.instance.Network().HostName(), t.port)
}
