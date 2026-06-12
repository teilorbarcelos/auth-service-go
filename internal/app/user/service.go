package user

import (
	"context"
	"errors"
	"io"
	"time"

	"backend-go/internal/core/models"
	"backend-go/internal/infra/pdf"
	"backend-go/internal/infra/session"
	"backend-go/pkg/config"
	"backend-go/pkg/database"
	"backend-go/pkg/logger"
	"backend-go/pkg/security"
	"go.uber.org/zap"
)

const roleNameFilter = "Role.name"

type UserRepositoryI interface {
	Create(user *models.User) error
	Update(id string, updates map[string]interface{}) error
	Delete(id string) error
	FindByID(id string, preloads ...string) (*models.User, error)
	FindByEmail(email string, preloads ...string) (*models.User, error)
	UpdatePassword(authID string, password string) error
	SearchPaginated(params database.FilterParams, filterable map[string]database.FilterConfig, searchable []database.SearchConfig, preloads ...string) ([]models.User, int64, error)
	WithContext(ctx context.Context) UserRepositoryI
	IncrementSessionVersion(ctx context.Context, userID string) (int, error)
}

func checkAdminUserUpdate(user *models.User, dto UpdateUserDTO) error {
	if dto.Active != nil && !*dto.Active {
		return errors.New("o usuário administrador inicial não pode ser desativado")
	}
	if dto.Email != "" && dto.Email != user.Email {
		return errors.New("o email do usuário administrador inicial não pode ser alterado")
	}
	return nil
}

func buildUserUpdates(user *models.User, dto UpdateUserDTO) (map[string]interface{}, error) {
	updates := make(map[string]interface{})

	if user.Email == config.AppConfig.FirstUserEmail {
		if err := checkAdminUserUpdate(user, dto); err != nil {
			return nil, err
		}
	} else {
		if dto.Email != "" {
			updates["email"] = dto.Email
		}
		if dto.Active != nil {
			updates["active"] = *dto.Active
		}
	}

	if dto.Name != "" {
		updates["name"] = dto.Name
	}
	if dto.IDRole != "" {
		updates["id_role"] = dto.IDRole
	}
	return updates, nil
}

type UserService struct {
	Repo           UserRepositoryI
	SessionManager session.SessionStore
	PdfProvider    pdf.PdfProvider
}

func NewUserService(repo UserRepositoryI, sessionMgr session.SessionStore, pdfProvider pdf.PdfProvider) *UserService {
	return &UserService{
		Repo:           repo,
		SessionManager: sessionMgr,
		PdfProvider:    pdfProvider,
	}
}

type CreateUserDTO struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	IDRole   string `json:"id_role" binding:"required"`
	Active   bool   `json:"active"`
}

type UpdateUserDTO struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	IDRole   string `json:"id_role"`
	Active   *bool  `json:"active"`
}

func (s *UserService) Create(ctx context.Context, dto CreateUserDTO) (*models.User, error) {
	hashedPassword, err := security.HashPassword(dto.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Name:   dto.Name,
		Email:  dto.Email,
		Active: true,
		IDRole: dto.IDRole,
		Auth: &models.Auth{
			Password: &hashedPassword,
			Active:   true,
		},
	}

	err = s.Repo.WithContext(ctx).Create(user)
	return user, err
}

func (s *UserService) Update(ctx context.Context, id string, dto UpdateUserDTO) (*models.User, error) {
	repo := s.Repo.WithContext(ctx)
	user, err := repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	updates, err := buildUserUpdates(user, dto)
	if err != nil {
		return nil, err
	}

	if len(updates) > 0 {
		if err := repo.Update(id, updates); err != nil {
			return nil, err
		}
	}

	if dto.Password != "" {
		hashedPassword, _ := security.HashPassword(dto.Password)
		if user.IDAuth != nil {
			if err := repo.UpdatePassword(*user.IDAuth, hashedPassword); err != nil {
				return nil, err
			}
		}
	}

	s.bumpSessionVersion(ctx, id)

	return repo.FindByID(id, "Auth", "Role")
}

func (s *UserService) List(ctx context.Context, params database.FilterParams) ([]models.User, int64, error) {
	filterable := map[string]database.FilterConfig{
		"name":         {Operator: "contains"},
		"email":        {Operator: "equals"},
		"active":       {Type: "boolean"},
		"created_at":   {Type: "date"},
		"updated_at":   {Type: "date"},
		roleNameFilter: {Relation: "nested"},
	}

	searchable := []database.SearchConfig{
		{Key: "name"},
		{Key: "email"},
		{Key: roleNameFilter, Relation: "nested"},
	}

	return s.Repo.WithContext(ctx).SearchPaginated(params, filterable, searchable)
}

func (s *UserService) GetByID(ctx context.Context, id string) (*models.User, error) {
	return s.Repo.WithContext(ctx).FindByID(id, "Auth", "Role")
}

func (s *UserService) Delete(ctx context.Context, id string) error {
	user, err := s.Repo.WithContext(ctx).FindByID(id)
	if err != nil {
		return err
	}
	if user.Email == config.AppConfig.FirstUserEmail {
		return errors.New("o usuário administrador inicial não pode ser excluído")
	}
	// LGPD User Anonymization
	updates := map[string]interface{}{
		"name":  "Deleted User",
		"email": "deleted-" + id + "@anonymized.local",
	}
	if err := s.Repo.WithContext(ctx).Update(id, updates); err != nil {
		return err
	}

	if err := s.Repo.WithContext(ctx).Delete(id); err != nil {
		return err
	}
	s.bumpSessionVersion(ctx, id)
	return nil
}

func (s *UserService) SetStatus(ctx context.Context, id string, active bool) error {
	user, err := s.Repo.WithContext(ctx).FindByID(id)
	if err != nil {
		return err
	}
	if user.Email == config.AppConfig.FirstUserEmail && !active {
		return errors.New("o usuário administrador inicial não pode ser desativado")
	}
	if err := s.Repo.WithContext(ctx).Update(id, map[string]interface{}{"active": active}); err != nil {
		return err
	}
	s.bumpSessionVersion(ctx, id)
	return nil
}

func (s *UserService) bumpSessionVersion(ctx context.Context, id string) {
	newVersion, err := s.Repo.WithContext(ctx).IncrementSessionVersion(ctx, id)
	if err != nil {
		logger.Warn("failed to bump session version", zap.String("userID", id), zap.Error(err))
		return
	}
	s.SessionManager.SetSessionVersion(ctx, id, newVersion)
}

func (s *UserService) ExportPdf(ctx context.Context, params database.FilterParams) (io.ReadCloser, error) {
	filterable := map[string]database.FilterConfig{
		"name":         {Operator: "contains"},
		"email":        {Operator: "equals"},
		"active":       {Type: "boolean"},
		"created_at":   {Type: "date"},
		"updated_at":   {Type: "date"},
		roleNameFilter: {Relation: "nested"},
	}

	searchable := []database.SearchConfig{
		{Key: "name"},
		{Key: "email"},
		{Key: roleNameFilter, Relation: "nested"},
	}

	users, _, err := s.Repo.WithContext(ctx).SearchPaginated(params, filterable, searchable, "Role")
	if err != nil {
		return nil, err
	}

	usersData := make([]map[string]interface{}, 0, len(users))
	for _, u := range users {
		var roleName *string
		if u.Role != nil {
			roleName = &u.Role.Name
		}

		var phone *string
		if u.Phone != nil {
			phone = u.Phone
		}

		usersData = append(usersData, map[string]interface{}{
			"id":       u.ID,
			"name":     u.Name,
			"email":    u.Email,
			"phone":    phone,
			"roleName": roleName,
			"active":   u.Active,
		})
	}

	localTime := time.Now().Format("02/01/2006 15:04:05")

	pdfData := map[string]interface{}{
		"title":       "Relatório de Usuários",
		"generatedAt": localTime,
		"users":       usersData,
	}

	request := pdf.PdfRequestDTO{
		Template: "user-list",
		Data:     pdfData,
	}

	return s.PdfProvider.GeneratePdf(request)
}
