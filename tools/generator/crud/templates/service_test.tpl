package {{.LowerName}}

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"backend-go/pkg/database"
)

func Test{{.Name}}Service_Create(t *testing.T) {
	repo := New{{.Name}}Repository(database.DB)
	service := New{{.Name}}Service(repo)
	ctx := context.Background()

	dto := Create{{.Name}}DTO{
		Name: "Service Test",
	}

	entity, err := service.Create(ctx, dto)
	assert.NoError(t, err)
	assert.NotNil(t, entity)
	assert.Equal(t, dto.Name, entity.Name)
}

func Test{{.Name}}Service_Update(t *testing.T) {
	repo := New{{.Name}}Repository(database.DB)
	service := New{{.Name}}Service(repo)
	ctx := context.Background()

	c, _ := service.Create(ctx, Create{{.Name}}DTO{Name: "C"})

	t.Run("Success", func(t *testing.T) {
		res, err := service.Update(ctx, c.ID, map[string]interface{}{"name": "N"})
		assert.NoError(t, err)
		assert.Equal(t, "N", res.Name)
	})

	t.Run("Error - Database Constraint", func(t *testing.T) {
		_, err := service.Update(ctx, c.ID, map[string]interface{}{"name": nil})
		assert.Error(t, err)
	})
}

func Test{{.Name}}Service_List(t *testing.T) {
	repo := New{{.Name}}Repository(database.DB)
	service := New{{.Name}}Service(repo)
	ctx := context.Background()

	params := database.FilterParams{
		Pagination: database.Pagination{
			Page:  1,
			Limit: 10,
		},
	}

	items, total, err := service.List(ctx, params)
	assert.NoError(t, err)
	assert.NotNil(t, items)
	assert.True(t, total >= 0)
}

func Test{{.Name}}Service_GetByID(t *testing.T) {
	repo := New{{.Name}}Repository(database.DB)
	service := New{{.Name}}Service(repo)
	ctx := context.Background()

	c, _ := service.Create(ctx, Create{{.Name}}DTO{Name: "G"})

	res, err := service.GetByID(ctx, c.ID)
	assert.NoError(t, err)
	assert.Equal(t, c.ID, res.ID)
}

func Test{{.Name}}Service_Delete(t *testing.T) {
	repo := New{{.Name}}Repository(database.DB)
	service := New{{.Name}}Service(repo)
	ctx := context.Background()

	c, _ := service.Create(ctx, Create{{.Name}}DTO{Name: "D"})

	err := service.Delete(ctx, c.ID)
	assert.NoError(t, err)

	_, err = service.GetByID(ctx, c.ID)
	assert.Error(t, err)
}

func Test{{.Name}}Service_SetStatus(t *testing.T) {
	repo := New{{.Name}}Repository(database.DB)
	service := New{{.Name}}Service(repo)
	ctx := context.Background()

	c, _ := service.Create(ctx, Create{{.Name}}DTO{Name: "S"})

	err := service.SetStatus(ctx, c.ID, false)
	assert.NoError(t, err)

	res, _ := service.GetByID(ctx, c.ID)
	assert.False(t, res.Active)
}
