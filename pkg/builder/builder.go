package builder

import "context"

type Builder interface {
	Build(ctx context.Context, b *BuilderOptions) (logs string, err error)
}

type CacheOptions struct {
	Enabled bool
	Dir     string
	Repo    string
}

type BuilderOptions struct {
	ImageName    string
	BuildContext string
	Args         []string
	Destination  string
	Cache        *CacheOptions
}
