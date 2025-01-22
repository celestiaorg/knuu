package services

import "github.com/celestiaorg/knuu/pkg/errors"

type Error = errors.Error

var (
	ErrUsernameAlreadyTaken = errors.New("UsernameAlreadyTaken", "username already taken")
	ErrUserNotFound         = errors.New("UserNotFound", "user not found")
	ErrCreatingAdminUser    = errors.New("CreatingAdminUser", "error creating admin user")
	ErrUserIDRequired       = errors.New("UserIDRequired", "user ID is required")
	ErrTestAlreadyExists    = errors.New("TestAlreadyExists", "test already exists")
	ErrTestNotFound         = errors.New("TestNotFound", "test not found")
	ErrInvalidCredentials   = errors.New("InvalidCredentials", "invalid credentials")
	ErrScopeRequired        = errors.New("ScopeRequired", "scope is required")
)
