package repos

import (
	"context"

	"github.com/celestiaorg/knuu/internal/database/models"

	"gorm.io/gorm"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	FindUserByUsername(ctx context.Context, username string) (*models.User, error)
	FindUserByID(ctx context.Context, id uint) (*models.User, error)
	UpdatePassword(ctx context.Context, id uint, password string) error
	DeleteUserById(ctx context.Context, id uint) error
}

type userRepositoryImpl struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepositoryImpl{db: db}
}

func (r *userRepositoryImpl) CreateUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepositoryImpl) FindUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where(&models.User{Username: username}).First(&user).Error
	return &user, err
}

func (r *userRepositoryImpl) FindUserByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where(&models.User{ID: id}).First(&user).Error
	return &user, err
}

func (r *userRepositoryImpl) UpdatePassword(ctx context.Context, id uint, password string) error {
	updatedUser := &models.User{
		Password: password,
	}
	return r.db.WithContext(ctx).Model(&models.User{}).
		Where(&models.User{ID: id}).Updates(updatedUser).Error
}

func (r *userRepositoryImpl) DeleteUserById(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.User{ID: id}).Error
}
