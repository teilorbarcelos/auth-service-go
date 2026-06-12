package {{.LowerName}}

import (
	"context"
	"backend-go/internal/core/models"
	"backend-go/internal/core/repository"
	"gorm.io/gorm"
)

type {{.Name}}Repository struct {
	repository.BaseRepository[models.{{.Name}}]
}

func (r *{{.Name}}Repository) WithContext(ctx context.Context) *{{.Name}}Repository {
	return &{{.Name}}Repository{
		BaseRepository: *r.BaseRepository.WithContext(ctx),
	}
}

func New{{.Name}}Repository(db *gorm.DB) *{{.Name}}Repository {
	return &{{.Name}}Repository{
		BaseRepository: *repository.NewBaseRepository[models.{{.Name}}](db),
	}
}
