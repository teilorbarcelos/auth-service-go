package auth

import (
	"context"
	"testing"

	"backend-go/internal/core/models"
	"backend-go/pkg/database"
	"github.com/stretchr/testify/assert"
)

func TestNewAuthRepository(t *testing.T) {
	repo := NewRepository(database.DB)
	assert.NotNil(t, repo)
}

func TestAuthRepository_FindByEmail(t *testing.T) {
	repo := NewRepository(database.DB)
	ctx := context.Background()

	// Setup: Create Role, Auth and User
	feature := models.Feature{
		Name:        "Test Feature",
		Description: "Test Feature Description",
	}
	database.DB.Create(&feature)

	role := models.Role{
		Name: "Test Role",
		RoleFeature: []models.RoleFeature{
			{IDFeature: feature.ID},
		},
	}
	database.DB.Create(&role)

	password := "hashedpassword"
	auth := models.Auth{
		Password: &password,
	}
	database.DB.Create(&auth)

	user := models.User{
		Name:   "Auth Test User",
		Email:  "authtest@example.com",
		IDRole: role.ID,
		IDAuth: &auth.ID,
	}
	database.DB.Create(&user)

	t.Run("Success", func(t *testing.T) {
		found, err := repo.FindByEmail(ctx, user.Email)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, user.ID, found.ID)

		// Verify preloads
		assert.NotNil(t, found.Auth)
		assert.Equal(t, *auth.Password, *found.Auth.Password)
		assert.NotNil(t, found.Role)
		assert.Equal(t, role.ID, found.Role.ID)
		assert.NotEmpty(t, found.Role.RoleFeature)
	})

	t.Run("Not Found", func(t *testing.T) {
		found, err := repo.FindByEmail(ctx, "nonexistent@example.com")
		assert.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestAuthRepository_UpdateAuth(t *testing.T) {
	repo := NewRepository(database.DB)
	ctx := context.Background()

	auth := models.Auth{}
	database.DB.Create(&auth)

	t.Run("Success", func(t *testing.T) {
		token := "654321"
		updates := map[string]interface{}{
			"request_password_token": token,
		}

		err := repo.UpdateAuth(ctx, auth.ID, updates)
		assert.NoError(t, err)

		var updated models.Auth
		database.DB.First(&updated, "id = ?", auth.ID)
		assert.Equal(t, token, *updated.RequestPasswordToken)
	})
}
