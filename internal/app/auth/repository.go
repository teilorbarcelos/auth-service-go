package auth

import (
	"context"

	"backend-go/internal/core/models"
	"backend-go/internal/core/repository"
	"gorm.io/gorm"
)

type Repository interface {
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateAuth(ctx context.Context, authID string, updates map[string]interface{}) error
}

type authRepository struct {
	repository.BaseRepository[models.User]
}

func NewRepository(db *gorm.DB) Repository {
	return &authRepository{
		BaseRepository: *repository.NewBaseRepository[models.User](db),
	}
}

func (r *authRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.DB.WithContext(ctx).
		Preload("Auth").
		Preload("Role").
		Preload("Role.RoleFeature").
		Where("email = ?", email).
		First(&user).Error

	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) UpdateAuth(ctx context.Context, authID string, updates map[string]interface{}) error {
	return r.DB.WithContext(ctx).Model(&models.Auth{}).Where("id = ?", authID).Updates(updates).Error
}
