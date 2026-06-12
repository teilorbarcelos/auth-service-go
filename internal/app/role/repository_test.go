package role

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"backend-go/internal/core/models"
	"backend-go/pkg/database"
	"gorm.io/gorm"
)


func TestRoleRepository_Create(t *testing.T) {
	repo := NewRoleRepository(database.DB)
	role := &models.Role{
		Name:        "Test Role",
		Description: "Test Description",
		Active:      true,
	}

	err := repo.Create(role)
	assert.NoError(t, err)
	assert.NotEmpty(t, role.ID)
}

func TestRoleRepository_FindByID(t *testing.T) {
	repo := NewRoleRepository(database.DB)
	role := &models.Role{
		Name:        "Find Test",
		Description: "Find Description",
	}
	repo.Create(role)

	found, err := repo.FindByID(role.ID)
	assert.NoError(t, err)
	assert.Equal(t, role.ID, found.ID)
}

func TestRoleRepository_CreateWithPermissions(t *testing.T) {
	repo := NewRoleRepository(database.DB)
	
	t.Run("Success", func(t *testing.T) {
		role := &models.Role{Name: "With Perms", Description: "Desc"}
		database.DB.Create(&models.Feature{BaseModel: models.BaseModel{ID: "feat1"}, Name: "F1", Description: "D"})
		perms := []models.RoleFeature{
			{IDFeature: "feat1", View: true},
		}
		err := repo.CreateWithPermissions(role, perms)
		assert.NoError(t, err)
		assert.NotEmpty(t, role.ID)
	})

	t.Run("Error - ID Collision", func(t *testing.T) {
		role1 := &models.Role{Name: "R1", Description: "D"}
		role1.ID = "fixed-id"
		repo.Create(role1)

		role2 := &models.Role{Name: "R2", Description: "D"}
		role2.ID = "fixed-id"
		err := repo.CreateWithPermissions(role2, nil)
		assert.Error(t, err)
	})

	t.Run("Error - Permission Violation", func(t *testing.T) {
		role := &models.Role{Name: "Perm Error", Description: "D"}
		database.DB.Create(&models.Feature{BaseModel: models.BaseModel{ID: "feat_same"}, Name: "FS", Description: "D"})
		perms := []models.RoleFeature{
			{IDFeature: "feat_same", View: true},
			{IDFeature: "feat_same", View: true}, // Duplicate PK
		}
		err := repo.CreateWithPermissions(role, perms)
		assert.Error(t, err)
	})

	t.Run("Success - Nil Permissions", func(t *testing.T) {
		role := &models.Role{Name: "Nil Perms", Description: "D"}
		err := repo.CreateWithPermissions(role, nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, role.ID)
	})
}

func TestRoleRepository_UpdateWithPermissions(t *testing.T) {
	repo := NewRoleRepository(database.DB)
	role := &models.Role{Name: "To Update", Description: "D"}
	repo.Create(role)

	t.Run("Success", func(t *testing.T) {
		role.Name = "Updated Name"
		database.DB.Create(&models.Feature{BaseModel: models.BaseModel{ID: "feat2"}, Name: "F2", Description: "D"})
		perms := []models.RoleFeature{
			{IDFeature: "feat2", View: true},
		}
		err := repo.UpdateWithPermissions(role.ID, role, perms)
		assert.NoError(t, err)
	})

	t.Run("Error - Permission Violation", func(t *testing.T) {
		database.DB.Create(&models.Feature{BaseModel: models.BaseModel{ID: "f1"}, Name: "F1", Description: "D"})
		perms := []models.RoleFeature{
			{IDFeature: "f1", View: true},
			{IDFeature: "f1", View: true}, // Duplicate
		}
		err := repo.UpdateWithPermissions(role.ID, role, perms)
		assert.Error(t, err)
	})

	t.Run("Success - Empty Permissions", func(t *testing.T) {
		err := repo.UpdateWithPermissions(role.ID, role, []models.RoleFeature{})
		assert.NoError(t, err)
	})

	t.Run("Success - Nil Permissions", func(t *testing.T) {
		err := repo.UpdateWithPermissions(role.ID, role, nil)
		assert.NoError(t, err)
	})

	t.Run("Error - Name Too Long", func(t *testing.T) {
		invalidRole := &models.Role{Name: string(make([]byte, 300))}
		err := repo.UpdateWithPermissions(role.ID, invalidRole, nil)
		assert.Error(t, err)
	})

	t.Run("Error - ID Too Long", func(t *testing.T) {
		invalidID := string(make([]byte, 500))
		tempRole := &models.Role{Name: role.Name, Description: role.Description}
		err := repo.UpdateWithPermissions(invalidID, tempRole, nil)
		assert.Error(t, err)
	})

	t.Run("Error - Delete Violation", func(t *testing.T) {
		// Add a hook that always returns an error for Delete operations on role_feature
		database.DB.Callback().Delete().Before("gorm:delete").Register("test:error", func(db *gorm.DB) {
			if db.Statement.Schema != nil && db.Statement.Schema.Table == "role_feature" {
				db.AddError(fmt.Errorf("forced delete error"))
			}
		})
		defer database.DB.Callback().Delete().Remove("test:error")

		err := repo.UpdateWithPermissions(role.ID, role, []models.RoleFeature{{IDFeature: "f_err"}})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forced delete error")
	})
}

func TestRoleRepository_ListFeatures(t *testing.T) {
	repo := NewRoleRepository(database.DB)
	ctx := context.Background()

	database.DB.Create(&models.Feature{BaseModel: models.BaseModel{ID: "f_list"}, Name: "F List", Description: "D", Active: true})

	features, err := repo.ListFeatures(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, features)
}
