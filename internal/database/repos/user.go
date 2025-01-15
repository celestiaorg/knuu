package repos

import (
	"github.com/celestiaorg/knuu/internal/database/models"

	"gorm.io/gorm"
)

type UserRepository interface {
	CreateUser(user *models.User) error
	FindUserByUsername(username string) (*models.User, error)
	FindUserByID(id uint) (*models.User, error)
	UpdatePassword(id uint, password string) error
	DeleteUserById(id uint) error
}

type userRepositoryImpl struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepositoryImpl{db: db}
}

func (r *userRepositoryImpl) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *userRepositoryImpl) FindUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where(&models.User{Username: username}).First(&user).Error
	return &user, err
}

func (r *userRepositoryImpl) FindUserByID(id uint) (*models.User, error) {
	var user models.User
	err := r.db.Where(&models.User{ID: id}).First(&user).Error
	return &user, err
}

func (r *userRepositoryImpl) UpdatePassword(id uint, password string) error {
	updatedUser := &models.User{
		Password: password,
	}
	return r.db.Model(&models.User{}).
		Where(&models.User{ID: id}).Updates(updatedUser).Error
}

func (r *userRepositoryImpl) DeleteUserById(id uint) error {
	return r.db.Delete(&models.User{ID: id}).Error
}
