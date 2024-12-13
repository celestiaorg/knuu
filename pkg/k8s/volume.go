package k8s

import (
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
)

type Volume struct {
	Path  string
	Size  resource.Quantity
	Owner int64
	files []*File
}

type File struct {
	Source     string
	Dest       string
	Chown      string
	Permission string
}

func (c *Client) NewFile(source, dest, chown, permission string) *File {
	return &File{
		Source:     source,
		Dest:       dest,
		Chown:      chown,
		Permission: permission,
	}
}

func (c *Client) NewVolume(path string, size resource.Quantity, owner int64) *Volume {
	return &Volume{
		Path:  path,
		Size:  size,
		Owner: owner,
	}
}

func (v *Volume) AddFile(f *File) error {
	if err := validateFile(f); err != nil {
		return err
	}

	ok, err := v.isSubpath(f.Dest)
	if err != nil {
		return err
	}
	if !ok {
		return ErrDestNotSubpath.WithParams(f.Dest, v.Path)
	}

	v.files = append(v.files, f)
	return nil
}

func (v *Volume) Files() []*File {
	return v.files
}

func (v *Volume) isSubpath(target string) (bool, error) {
	base := filepath.Clean(v.Path)
	target = filepath.Clean(target)

	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false, err
	}

	return !strings.HasPrefix(rel, ".."), nil
}
