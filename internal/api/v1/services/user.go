package services

import (
	"context"

	"golang.org/x/crypto/bcrypt"

	"github.com/celestiaorg/knuu/internal/database/models"
	"github.com/celestiaorg/knuu/internal/database/repos"
)

type UserService interface {
	Register(ctx context.Context, user *models.User) (*models.User, error)
	Authenticate(ctx context.Context, username, password string) (*models.User, error)
}

type userServiceImpl struct {
	repo repos.UserRepository
}

var _ UserService = &userServiceImpl{}

// This function is used to create the admin user and the user service.
// It is called when the API is initialized.
func NewUserService(ctx context.Context, adminUser, adminPass string, userRepo repos.UserRepository) (UserService, error) {
	us := &userServiceImpl{
		repo: userRepo,
	}

	_, err := us.Register(ctx,
		&models.User{
			Username: adminUser,
			Password: adminPass,
			Role:     models.RoleAdmin,
		})
	if err != nil && err != ErrUsernameAlreadyTaken {
		return nil, ErrCreatingAdminUser.Wrap(err)
	}

	return us, nil
}

func (s *userServiceImpl) Register(ctx context.Context, user *models.User) (*models.User, error) {
	if _, err := s.repo.FindUserByUsername(ctx, user.Username); err == nil {
		return nil, ErrUsernameAlreadyTaken
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user.Password = string(hashedPassword)
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userServiceImpl) Authenticate(ctx context.Context, username, password string) (*models.User, error) {
	user, err := s.repo.FindUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, err
	}

	return user, nil
}
