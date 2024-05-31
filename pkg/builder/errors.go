package builder

import (
	"github.com/celestiaorg/knuu/pkg/errors"
)

type Error = errors.Error

var (
	ErrBuildContextEmpty = errors.New("BuildContextEmpty", "build context cannot be empty")
)
