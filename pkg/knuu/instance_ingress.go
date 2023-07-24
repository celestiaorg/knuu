package knuu

import "fmt"

type Ingress struct {
	Host               string // Required
	Path               string // Required
	PathType           string // Required
	Port               int    // Required
	BackendProtocol    string
	TlsEnabled         bool
	CertManagerEnabled bool
	SslPassthrough     bool
	ForceSslRedirect   bool
	EnableCors         bool
}

func (i Ingress) String() string {
	return fmt.Sprintf("Ingress{Host: %s, Path: %s, PathType: %s, Port: %d, BackendProtocol: %s, TlsEnabled: %t, CertManagerEnabled: %t, SslPassthrough: %t, ForceSslRedirect: %t, EnableCors: %t}", i.Host, i.Path, i.PathType, i.Port, i.BackendProtocol, i.TlsEnabled, i.CertManagerEnabled, i.SslPassthrough, i.ForceSslRedirect, i.EnableCors)
}

func (i Ingress) validate() error {
	if i.Host == "" {
		return fmt.Errorf("host is required")
	}
	if i.Path == "" {
		return fmt.Errorf("path is required")
	}
	if i.PathType == "" {
		return fmt.Errorf("path type is required")
	}
	if i.Port == 0 {
		return fmt.Errorf("port is required")
	}
	return nil
}
