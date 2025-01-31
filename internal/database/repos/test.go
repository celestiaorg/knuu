package repos

import (
	"context"

	"gorm.io/gorm"

	"github.com/celestiaorg/knuu/internal/database/models"
)

type TestRepository struct {
	db *gorm.DB
}

func NewTestRepository(db *gorm.DB) *TestRepository {
	return &TestRepository{
		db: db,
	}
}

func (r *TestRepository) Create(ctx context.Context, test *models.Test) error {
	return r.db.WithContext(ctx).Create(test).Error
}

func (r *TestRepository) Get(ctx context.Context, userID uint, scope string) (*models.Test, error) {
	var test models.Test
	err := r.db.WithContext(ctx).Where(&models.Test{UserID: userID, Scope: scope}).First(&test).Error
	return &test, err
}

func (r *TestRepository) Delete(ctx context.Context, scope string) error {
	return r.db.WithContext(ctx).Delete(&models.Test{Scope: scope}).Error
}

func (r *TestRepository) Update(ctx context.Context, test *models.Test) error {
	return r.db.WithContext(ctx).Model(&models.Test{}).Where(&models.Test{Scope: test.Scope, UserID: test.UserID}).Updates(test).Error
}

func (r *TestRepository) List(ctx context.Context, userID uint, limit int, offset int) ([]models.Test, error) {
	var tests []models.Test
	err := r.db.WithContext(ctx).
		Where(&models.Test{UserID: userID}).
		Limit(limit).Offset(offset).
		Order(models.TestFinishedField + " ASC").
		Order(models.TestCreatedAtField + " DESC").
		Find(&tests).Error
	return tests, err
}

func (r *TestRepository) Count(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Test{}).Where(&models.Test{UserID: userID}).Count(&count).Error
	return count, err
}

func (r *TestRepository) ListAllAlive(ctx context.Context) ([]models.Test, error) {
	var tests []models.Test
	err := r.db.WithContext(ctx).Where(&models.Test{Finished: false}).Find(&tests).Error
	return tests, err
}
