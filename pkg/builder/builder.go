package builder

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type Builder interface {
	Build(ctx context.Context, b *BuilderOptions) (logs string, err error)
}

type BuilderOptions struct {
	ImageName    string
	BuildContext string
	Args         []ArgInterface
	Destination  string
	Cache        *CacheOptions
}

type CacheOptions struct {
	Enabled bool
	Dir     string
	Repo    string
}

func (c *CacheOptions) Default(buildContext string) (*CacheOptions, error) {
	if buildContext == "" {
		return nil, ErrBuildContextEmpty
	}

	ctxHash, err := hashString(buildContext)
	if err != nil {
		return nil, err
	}

	return &CacheOptions{
		Enabled: true,
		Dir:     "",
		// ttl.sh with the hash of build context is used as the cache repo
		// Kaniko adds a string tag to the image name, so we don't need to add it here
		Repo: fmt.Sprintf("ttl.sh/%s:24h", ctxHash),
	}, nil
}

func DefaultImageName(buildContext string) (string, error) {
	if buildContext == "" {
		return "", ErrBuildContextEmpty
	}

	ctxHash, err := hashString(buildContext)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("ttl.sh/%s:24h", ctxHash), nil
}

func hashString(s string) (string, error) {
	hash := sha256.New()
	if _, err := hash.Write([]byte(s)); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
