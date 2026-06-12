package user

import (
	"context"
	"backend-go/internal/core/models"
	"backend-go/internal/core/repository"
	"gorm.io/gorm"
)

type UserRepository struct {
	repository.BaseRepository[models.User]
}

func (r *UserRepository) WithContext(ctx context.Context) UserRepositoryI {
	return &UserRepository{
		BaseRepository: *r.BaseRepository.WithContext(ctx),
	}
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		BaseRepository: *repository.NewBaseRepository[models.User](db),
	}
}

func (r *UserRepository) FindByEmail(email string, preloads ...string) (*models.User, error) {
	var user models.User
	query := r.DB.Model(new(models.User))
	for _, p := range preloads {
		query = query.Preload(p)
	}
	err := query.Where("email = ? AND is_deleted = ?", email, false).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) UpdatePassword(authID string, password string) error {
	return r.DB.Model(&models.Auth{}).Where("id = ?", authID).Update("password", password).Error
}

func (r *UserRepository) IncrementSessionVersion(ctx context.Context, userID string) (int, error) {
	var version int
	err := r.DB.WithContext(ctx).Model(&models.Auth{}).
		Where("id = (SELECT id_auth FROM \"user\" WHERE id = ?)", userID).
		Update("session_version", gorm.Expr("session_version + 1")).
		Error
	if err != nil {
		return 0, err
	}
	r.DB.WithContext(ctx).Model(&models.Auth{}).
		Select("session_version").
		Where("id = (SELECT id_auth FROM \"user\" WHERE id = ?)", userID).
		Scan(&version)
	return version, nil
}

