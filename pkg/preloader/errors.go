package preloader

import (
	"github.com/celestiaorg/knuu/pkg/errors"
)

type Error = errors.Error

var (
	ErrGeneratingK8sNameForPreloader = errors.New("GeneratingK8sNameForPreloader", "error generating k8s name for preloader")
)
