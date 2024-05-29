package builder

import (
	"github.com/celestiaorg/knuu/pkg/errors"
)

type Error = errors.Error

var (
	ErrBuildContextEmpty = &Error{Code: "BuildContextEmpty", Message: "build context cannot be empty"}
)
