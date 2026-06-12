package auth

import (
	"context"

	"github.com/teilorbarcelos/auth-service-go/internal/core/models"
	"gorm.io/gorm"
)

type Repository interface {
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateAuth(ctx context.Context, authID string, updates map[string]interface{}) error
}

type authRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &authRepository{db: db}
}

func (r *authRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).
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
	return r.db.WithContext(ctx).Model(&models.Auth{}).Where("id = ?", authID).Updates(updates).Error
}
