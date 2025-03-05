package registry

import "github.com/celestiaorg/knuu/pkg/errors"

type Error = errors.Error

var (
	ErrRegionNamespaceUsernamePasswordRequired = errors.New("RegionNamespaceUsernamePasswordRequired", "region, namespace, username and password are required")
	ErrUsernamePasswordRequired                = errors.New("UsernamePasswordRequired", "username and password are required")
)
