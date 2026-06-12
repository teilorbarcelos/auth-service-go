package role

import (
	"backend-go/internal/core/models"
	"backend-go/internal/core/repository"
	"backend-go/pkg/database"
	"context"

	"gorm.io/gorm"
)

type RoleRepositoryI interface {
	WithContext(ctx context.Context) RoleRepositoryI
	Create(role *models.Role) error
	Delete(id string) error
	Update(id string, updates map[string]interface{}) error
	CreateWithPermissions(role *models.Role, permissions []models.RoleFeature) error
	UpdateWithPermissions(id string, role *models.Role, permissions []models.RoleFeature) error
	FindByID(id string, preloads ...string) (*models.Role, error)
	SearchPaginated(params database.FilterParams, filterable map[string]database.FilterConfig, searchable []database.SearchConfig, preloads ...string) ([]models.Role, int64, error)
	ListFeatures(ctx context.Context) ([]models.Feature, error)
	BulkIncrementSessionVersion(ctx context.Context, roleID string) error
	FindUserIDsByRole(ctx context.Context, roleID string) ([]string, error)
}

type RoleRepository struct {
	repository.BaseRepository[models.Role]
}

func (r *RoleRepository) WithContext(ctx context.Context) RoleRepositoryI {
	return &RoleRepository{
		BaseRepository: *r.BaseRepository.WithContext(ctx),
	}
}

func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{
		BaseRepository: *repository.NewBaseRepository[models.Role](db),
	}
}

func (r *RoleRepository) ListFeatures(ctx context.Context) ([]models.Feature, error) {
	var features []models.Feature
	err := r.DB.WithContext(ctx).Where("active = ?", true).Find(&features).Error
	return features, err
}

func (r *RoleRepository) CreateWithPermissions(role *models.Role, permissions []models.RoleFeature) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(role).Error; err != nil {
			return err
		}
		for i := range permissions {
			permissions[i].IDRole = role.ID
		}
		if len(permissions) > 0 {
			if err := tx.Create(&permissions).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func updatePermissions(tx *gorm.DB, id string, permissions []models.RoleFeature) error {
	if permissions == nil {
		return nil
	}
	if err := tx.Where("id_role = ?", id).Delete(&models.RoleFeature{}).Error; err != nil {
		return err
	}
	for i := range permissions {
		permissions[i].IDRole = id
	}
	if len(permissions) > 0 {
		if err := tx.Create(&permissions).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *RoleRepository) UpdateWithPermissions(id string, role *models.Role, permissions []models.RoleFeature) error {
	role.ID = id
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(role).Updates(role).Error; err != nil {
			return err
		}
		return updatePermissions(tx, id, permissions)
	})
}

func (r *RoleRepository) BulkIncrementSessionVersion(ctx context.Context, roleID string) error {
	return r.DB.WithContext(ctx).Exec(
		"UPDATE auth SET session_version = session_version + 1 WHERE id IN (SELECT id_auth FROM \"user\" WHERE id_role = ? AND is_deleted = false)",
		roleID,
	).Error
}

func (r *RoleRepository) FindUserIDsByRole(ctx context.Context, roleID string) ([]string, error) {
	var ids []string
	err := r.DB.WithContext(ctx).Model(&models.User{}).
		Select("id").
		Where("id_role = ? AND is_deleted = false", roleID).
		Find(&ids).Error
	return ids, err
}
