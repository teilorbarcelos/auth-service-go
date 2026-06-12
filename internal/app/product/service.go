package product

import (
	"backend-go/internal/core/models"
	"backend-go/pkg/database"
	"context"
)

type ProductService struct {
	Repo *ProductRepository
}

func NewProductService(repo *ProductRepository) *ProductService {
	return &ProductService{Repo: repo}
}

type CreateProductDTO struct {
	Name        string  `json:"name" binding:"required"`
	SKU         string  `json:"sku" binding:"required"`
	Category    string  `json:"category" binding:"required"`
	Price       float64 `json:"price" binding:"required"`
	Stock       int     `json:"stock"`
	Description string  `json:"description"`
}

func (s *ProductService) Create(ctx context.Context, dto CreateProductDTO) (*models.Product, error) {
	var userID *string
	if val := ctx.Value("userID"); val != nil {
		if id, ok := val.(string); ok && id != "" {
			userID = &id
		}
	}

	product := &models.Product{
		Name:        dto.Name,
		SKU:         dto.SKU,
		Category:    dto.Category,
		Price:       dto.Price,
		Stock:       dto.Stock,
		Description: dto.Description,
		Active:      true,
		IDUser:      userID,
	}
	err := s.Repo.WithContext(ctx).Create(product)
	return product, err
}

func (s *ProductService) Update(ctx context.Context, id string, updates map[string]interface{}) (*models.Product, error) {
	err := s.Repo.WithContext(ctx).Update(id, updates)
	if err != nil {
		return nil, err
	}
	return s.Repo.WithContext(ctx).FindByID(id)
}

func (s *ProductService) List(ctx context.Context, params database.FilterParams) ([]models.Product, int64, error) {
	filterable := map[string]database.FilterConfig{
		"name":       {Operator: "contains"},
		"sku":        {Operator: "equals"},
		"category":   {Operator: "equals"},
		"active":     {Type: "boolean"},
		"created_at": {Type: "date"},
		"updated_at": {Type: "date"},
	}

	searchable := []database.SearchConfig{
		{Key: "name"},
		{Key: "sku"},
		{Key: "category"},
		{Key: "description"},
	}

	return s.Repo.WithContext(ctx).SearchPaginated(params, filterable, searchable)
}

func (s *ProductService) GetByID(ctx context.Context, id string) (*models.Product, error) {
	return s.Repo.WithContext(ctx).FindByID(id)
}

func (s *ProductService) Delete(ctx context.Context, id string) error {
	return s.Repo.WithContext(ctx).Delete(id)
}

func (s *ProductService) SetStatus(ctx context.Context, id string, active bool) error {
	return s.Repo.WithContext(ctx).Update(id, map[string]interface{}{"active": active})
}
