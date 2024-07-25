package preloader

import (
	"github.com/celestiaorg/knuu/pkg/errors"
)

type Error = errors.Error

var (
	ErrPreloaderNameAlreadyExists = errors.New("PreloaderNameAlreadyExists", "preloader name '%s' already exists")
)
