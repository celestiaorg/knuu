package builder

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	DefaultRegistryAddress = "ttl.sh"
	DefaultImageTTL        = "24h" // used as a tag for the ephemeral images on ttl.sh
	DefaultImageTag        = "latest"
	DefaultCacheRepoName   = "cl-kaniko-cache" // only abcdefghijklmnopqrstuvwxyz0123456789_-./ are allowed
)

type Builder interface {
	Build(ctx context.Context, b BuilderOptions) (logs string, err error)
	ResolveImageName(buildContext string) (*ResolvedImage, error)
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

type ResolvedImage struct {
	Name     string
	Registry string
	Tag      string
}

func (r *ResolvedImage) ToString() string {
	return fmt.Sprintf("%s/%s:%s", r.Registry, r.Name, r.Tag)
}

func DefaultCacheOptions() *CacheOptions {
	return &CacheOptions{
		Enabled: true,
		Dir:     "",
		Repo:    ImageWithRegistry(DefaultCacheRepoName, DefaultRegistryAddress),
	}
}

func DefaultImageName(buildContext string) (string, error) {
	if buildContext == "" {
		return "", ErrBuildContextEmpty
	}

	return hashString(buildContext)
}

func IsImageWithTag(image string) bool {
	return strings.Contains(image, ":")
}

func IsImageWithRegistry(image string) bool {
	return strings.Contains(image, "/")
}

func ImageWithRegistry(name, registry string) string {
	return fmt.Sprintf("%s/%s", registry, name)
}

func ImageWithTag(name, tag string) string {
	return fmt.Sprintf("%s:%s", name, tag)
}

func hashString(s string) (string, error) {
	hash := sha256.New()
	if _, err := hash.Write([]byte(s)); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
