package builder

import (
	"strings"
)

const (
	dirProtocol = "dir:///"
)

type DirContext struct {
	Path string // This path must be an absolute path
}

func (d DirContext) BuildContext() string {
	return dirProtocol + strings.Trim(d.Path, "/")
}

func GetDirFromBuildContext(ctx string) string {
	// must be absolute path
	return "/" + strings.TrimPrefix(ctx, dirProtocol)
}

func IsDirContext(ctx string) bool {
	return strings.HasPrefix(ctx, dirProtocol)
}
