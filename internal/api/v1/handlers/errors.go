package handlers

import "github.com/celestiaorg/knuu/pkg/errors"

type Error = errors.Error

var (
	ErrInvalidCredentials = errors.New("InvalidCredentials", "invalid credentials")
)
