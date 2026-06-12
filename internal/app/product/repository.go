package product

import (
	"context"
	"backend-go/internal/core/models"
	"backend-go/internal/core/repository"
	"gorm.io/gorm"
)

type ProductRepository struct {
	repository.BaseRepository[models.Product]
}

func (r *ProductRepository) WithContext(ctx context.Context) *ProductRepository {
	return &ProductRepository{
		BaseRepository: *r.BaseRepository.WithContext(ctx),
	}
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{
		BaseRepository: *repository.NewBaseRepository[models.Product](db),
	}
}
