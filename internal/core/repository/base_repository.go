package repository

import (
	"context"
	"fmt"

	"backend-go/pkg/database"

	"gorm.io/gorm"
)

const idEqualsFilter = "id = ?"

type IRepository[T any] interface {
	Create(entity *T) error
	FindAll(filter map[string]interface{}, offset, limit int, preloads ...string) ([]T, int64, error)
	FindByID(id string, preloads ...string) (*T, error)
	Update(id string, updates map[string]interface{}) error
	Delete(id string) error     // Soft Delete
	HardDelete(id string) error // Destrói do banco
	SearchPaginated(params database.FilterParams, filterable map[string]database.FilterConfig, searchable []database.SearchConfig, preloads ...string) ([]T, int64, error)
}

type BaseRepository[T any] struct {
	DB *gorm.DB
}

func NewBaseRepository[T any](db *gorm.DB) *BaseRepository[T] {
	return &BaseRepository[T]{DB: db}
}

func (r *BaseRepository[T]) WithContext(ctx context.Context) *BaseRepository[T] {
	return &BaseRepository[T]{DB: r.DB.WithContext(ctx)}
}

func (r *BaseRepository[T]) Create(entity *T) error {
	return r.DB.Create(entity).Error
}

func (r *BaseRepository[T]) FindAll(filter map[string]interface{}, offset, limit int, preloads ...string) ([]T, int64, error) {
	var entities []T
	var total int64

	query := r.DB.Model(new(T)).Where("is_deleted = ?", false)

	for _, p := range preloads {
		query = query.Preload(p)
	}

	for k, v := range filter {
		query = query.Where(k+" = ?", v)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err = query.Find(&entities).Error
	return entities, total, err
}

func (r *BaseRepository[T]) FindByID(id string, preloads ...string) (*T, error) {
	var entity T
	query := r.DB.Model(new(T))
	for _, p := range preloads {
		query = query.Preload(p)
	}
	err := query.Where("id = ? AND is_deleted = ?", id, false).First(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *BaseRepository[T]) Update(id string, updates map[string]interface{}) error {
	if updates != nil {
		updates["id"] = id
	}
	return r.DB.Model(new(T)).Where(idEqualsFilter, id).Updates(updates).Error
}

func (r *BaseRepository[T]) Delete(id string) error {
	return r.DB.Model(new(T)).Where(idEqualsFilter, id).Updates(map[string]interface{}{
		"is_deleted": true,
	}).Delete(new(T)).Error
}

func (r *BaseRepository[T]) HardDelete(id string) error {
	return r.DB.Unscoped().Where(idEqualsFilter, id).Delete(new(T)).Error
}

func (r *BaseRepository[T]) SearchPaginated(params database.FilterParams, filterable map[string]database.FilterConfig, searchable []database.SearchConfig, preloads ...string) ([]T, int64, error) {
	var entities []T
	var total int64

	query := r.DB.Model(new(T))

	for _, p := range preloads {
		query = query.Preload(p)
	}

	if params.Filters == nil {
		params.Filters = make(map[string]interface{})
	}

	if params.Filters["ignoreDefaultFilters"] != true {
		if _, ok := filterable["active"]; ok {
			if _, exists := params.Filters["active"]; !exists {
				params.Filters["active"] = true
			}
		}
	}

	query, err := database.ApplyFilters(query, params, filterable, searchable)
	if err != nil {
		return nil, 0, err
	}
	if params.Filters["ignoreDefaultFilters"] != true {
		quotedField := "is_deleted"
		if query.Statement.Schema != nil {
			quotedField = query.Statement.Quote(query.Statement.Schema.Table + ".is_deleted")
		}
		query = query.Where(fmt.Sprintf("%s = ?", quotedField), false)
	}

	countQuery := query.Session(&gorm.Session{}).Offset(-1).Limit(-1)
	err = countQuery.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Find(&entities).Error
	return entities, total, err
}
