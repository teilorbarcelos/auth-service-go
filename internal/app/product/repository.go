package product

import (
	"context"
	"github.com/teilorbarcelos/auth-service-go/internal/core/models"
	"github.com/teilorbarcelos/auth-service-go/internal/core/repository"
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
