package role

import (
	"backend-go/internal/core/models"
	"backend-go/internal/infra/session"
	"backend-go/pkg/database"
	"backend-go/pkg/logger"
	"context"
	"go.uber.org/zap"
)

type RoleService struct {
	Repo           RoleRepositoryI
	SessionManager session.SessionStore
}

func NewRoleService(repo RoleRepositoryI, sessionMgr session.SessionStore) *RoleService {
	return &RoleService{
		Repo:           repo,
		SessionManager: sessionMgr,
	}
}

func (s *RoleService) ListFeatures(ctx context.Context) ([]models.Feature, error) {
	return s.Repo.WithContext(ctx).ListFeatures(ctx)
}

type CreateRoleDTO struct {
	Name        string               `json:"name" binding:"required"`
	Description string               `json:"description" binding:"required"`
	Permissions []models.RoleFeature `json:"permissions"`
}

func (s *RoleService) Create(ctx context.Context, dto CreateRoleDTO) (*models.Role, error) {
	role := &models.Role{
		Name:        dto.Name,
		Description: dto.Description,
		Active:      true,
	}

	err := s.Repo.WithContext(ctx).CreateWithPermissions(role, dto.Permissions)
	return role, err
}

func (s *RoleService) Update(ctx context.Context, id string, dto CreateRoleDTO) (*models.Role, error) {
	role := &models.Role{
		Name:        dto.Name,
		Description: dto.Description,
	}
	if err := s.Repo.WithContext(ctx).UpdateWithPermissions(id, role, dto.Permissions); err != nil {
		return nil, err
	}
	s.SessionManager.InvalidateRoleSessions(id)
	s.bulkBumpSessionVersion(ctx, id)
	return s.Repo.WithContext(ctx).FindByID(id, "RoleFeature")
}

func (s *RoleService) List(ctx context.Context, params database.FilterParams) ([]models.Role, int64, error) {
	filterable := map[string]database.FilterConfig{
		"name":       {Operator: "contains"},
		"active":     {Type: "boolean"},
		"created_at": {Type: "date"},
		"updated_at": {Type: "date"},
	}

	searchable := []database.SearchConfig{
		{Key: "name"},
	}

	return s.Repo.WithContext(ctx).SearchPaginated(params, filterable, searchable)
}

func (s *RoleService) GetByID(ctx context.Context, id string) (*models.Role, error) {
	return s.Repo.WithContext(ctx).FindByID(id, "RoleFeature")
}

func (s *RoleService) Delete(ctx context.Context, id string) error {
	if err := s.Repo.WithContext(ctx).Delete(id); err != nil {
		return err
	}
	s.SessionManager.InvalidateRoleSessions(id)
	s.bulkBumpSessionVersion(ctx, id)
	return nil
}

func (s *RoleService) bulkBumpSessionVersion(ctx context.Context, roleID string) {
	if err := s.Repo.WithContext(ctx).BulkIncrementSessionVersion(ctx, roleID); err != nil {
		logger.Warn("failed to bulk bump session version for role", zap.String("roleID", roleID), zap.Error(err))
		return
	}

	userIDs, err := s.Repo.WithContext(ctx).FindUserIDsByRole(ctx, roleID)
	if err != nil {
		logger.Warn("failed to find users for role version sync", zap.String("roleID", roleID), zap.Error(err))
		return
	}

	for _, uid := range userIDs {
		if err := s.SessionManager.InvalidateUserSessions(uid, roleID); err != nil {
			logger.Warn("failed to sync redis session version", zap.String("userID", uid), zap.Error(err))
		}
	}
}

func (s *RoleService) SetStatus(ctx context.Context, id string, active bool) error {
	if err := s.Repo.WithContext(ctx).Update(id, map[string]interface{}{"active": active}); err != nil {
		return err
	}
	s.SessionManager.InvalidateRoleSessions(id)
	s.bulkBumpSessionVersion(ctx, id)
	return nil
}
