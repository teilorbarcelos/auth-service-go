package {{.LowerName}}

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"backend-go/internal/core/models"
	"backend-go/pkg/database"
)

func Test{{.Name}}Repository_Create(t *testing.T) {
	repo := New{{.Name}}Repository(database.DB)
	entity := &models.{{.Name}}{
		Name:   "Test Entity",
		Active: true,
	}

	err := repo.Create(entity)
	assert.NoError(t, err)
	assert.NotEmpty(t, entity.ID)
}

func Test{{.Name}}Repository_FindByID(t *testing.T) {
	repo := New{{.Name}}Repository(database.DB)
	entity := &models.{{.Name}}{
		Name: "Find Test",
	}
	repo.Create(entity)

	found, err := repo.FindByID(entity.ID)
	assert.NoError(t, err)
	assert.Equal(t, entity.ID, found.ID)
}
