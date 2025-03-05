package builder

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/celestiaorg/knuu/pkg/builder/registry"
)

const (
	DefaultImageTTL      = "24h"             // used as a tag for the ephemeral images on ttl.sh
	DefaultCacheRepoName = "cl-kaniko-cache" // only abcdefghijklmnopqrstuvwxyz0123456789_-./ are allowed
)

type Builder interface {
	Build(ctx context.Context, b BuilderOptions) (logs string, err error)
	DefaultImage(buildContext string) (*registry.ResolvedImage, error)
	CacheOptions() *CacheOptions
}

type BuilderOptions struct {
	ImageName    string // Custom image name (if provided by the user)
	BuildContext string
	Args         []ArgInterface
	Cache        *CacheOptions
}

type CacheOptions struct {
	Enabled bool
	Dir     string
	Repo    string
}

func DefaultImageName(buildContext string) (string, error) {
	if buildContext == "" {
		return "", ErrBuildContextEmpty
	}

	return hashString(buildContext)
}

func hashString(s string) (string, error) {
	hash := sha256.New()
	if _, err := hash.Write([]byte(s)); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
