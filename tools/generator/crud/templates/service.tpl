package {{.LowerName}}

import (
	"backend-go/internal/core/models"
	"backend-go/pkg/database"
	"context"
)

type {{.Name}}Service struct {
	Repo *{{.Name}}Repository
}

func New{{.Name}}Service(repo *{{.Name}}Repository) *{{.Name}}Service {
	return &{{.Name}}Service{Repo: repo}
}

type Create{{.Name}}DTO struct {
	Name string `json:"name" binding:"required"`
}

func (s *{{.Name}}Service) Create(ctx context.Context, dto Create{{.Name}}DTO) (*models.{{.Name}}, error) {
	entity := &models.{{.Name}}{
		Name:   dto.Name,
		Active: true,
	}
	err := s.Repo.WithContext(ctx).Create(entity)
	return entity, err
}

func (s *{{.Name}}Service) Update(ctx context.Context, id string, updates map[string]interface{}) (*models.{{.Name}}, error) {
	err := s.Repo.WithContext(ctx).Update(id, updates)
	if err != nil {
		return nil, err
	}
	return s.Repo.WithContext(ctx).FindByID(id)
}

func (s *{{.Name}}Service) List(ctx context.Context, params database.FilterParams) ([]models.{{.Name}}, int64, error) {
	filterable := map[string]database.FilterConfig{
		"name":   {Operator: "contains"},
		"active": {Type: "boolean"},
	}

	searchable := []database.SearchConfig{
		{Key: "name"},
	}

	return s.Repo.WithContext(ctx).SearchPaginated(params, filterable, searchable)
}

func (s *{{.Name}}Service) GetByID(ctx context.Context, id string) (*models.{{.Name}}, error) {
	return s.Repo.WithContext(ctx).FindByID(id)
}

func (s *{{.Name}}Service) Delete(ctx context.Context, id string) error {
	return s.Repo.WithContext(ctx).Delete(id)
}

func (s *{{.Name}}Service) SetStatus(ctx context.Context, id string, active bool) error {
	return s.Repo.WithContext(ctx).Update(id, map[string]interface{}{"active": active})
}
