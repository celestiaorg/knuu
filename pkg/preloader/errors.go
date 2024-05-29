package preloader

import (
	"github.com/celestiaorg/knuu/pkg/errors"
)

type Error = errors.Error

var (
	ErrGeneratingK8sNameForPreloader = &Error{Code: "GeneratingK8sNameForPreloader", Message: "error generating k8s name for preloader"}
)
