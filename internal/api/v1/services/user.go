package services

import (
	"errors"
	"time"

	"github.com/celestiaorg/knuu/internal/database/models"
	"github.com/celestiaorg/knuu/internal/database/repos"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

const (
	UserTokenDuration = 1 * time.Hour
)

type UserService interface {
	Register(user *models.User) (*models.User, error)
	Authenticate(username, password string) (string, error)
}

type userServiceImpl struct {
	secretKey string
	userRepo  repos.UserRepository
}

var _ UserService = &userServiceImpl{}

// TODO: need to add the admin user for the first time
func NewUserService(secretKey string, userRepo repos.UserRepository) UserService {
	return &userServiceImpl{
		secretKey: secretKey,
		userRepo:  userRepo,
	}
}

func (s *userServiceImpl) Register(user *models.User) (*models.User, error) {
	if _, err := s.userRepo.FindUserByUsername(user.Username); err == nil {
		return nil, errors.New("username already taken")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user.Password = string(hashedPassword)
	if err := s.userRepo.CreateUser(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userServiceImpl) Authenticate(username, password string) (string, error) {
	user, err := s.userRepo.FindUserByUsername(username)
	if err != nil {
		return "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(UserTokenDuration).Unix(),
	})
	tokenString, err := token.SignedString([]byte(s.secretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
