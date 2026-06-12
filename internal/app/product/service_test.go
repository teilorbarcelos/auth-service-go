package product

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"backend-go/internal/core/models"
	"backend-go/pkg/database"
)

func TestProductService_Create(t *testing.T) {
	repo := NewProductRepository(database.DB)
	service := NewProductService(repo)

	t.Run("Create without userID", func(t *testing.T) {
		ctx := context.Background()
		dto := CreateProductDTO{
			Name:     "Service Test",
			SKU:      "SKU-SERVICE-NEW",
			Category: "Test",
			Price:    100.0,
		}
		product, err := service.Create(ctx, dto)
		assert.NoError(t, err)
		assert.NotNil(t, product)
		assert.Equal(t, dto.Name, product.Name)
		assert.Nil(t, product.IDUser)
	})

	t.Run("Create with userID", func(t *testing.T) {
		userID := "user-123-abc"

		// Create role and user to satisfy FK constraint
		role := &models.Role{
			Name:        "Administrator",
			Description: "Admin role",
		}
		role.ID = "administrator"
		database.DB.FirstOrCreate(role)

		user := &models.User{
			Name:   "Test User",
			Email:  "test-user-product@example.com",
			IDRole: "administrator",
		}
		user.ID = userID
		database.DB.FirstOrCreate(user)

		ctx := context.WithValue(context.Background(), "userID", userID)
		dto := CreateProductDTO{
			Name:     "Service Test 2",
			SKU:      "SKU-SERVICE-NEW-2",
			Category: "Test",
			Price:    100.0,
		}
		product, err := service.Create(ctx, dto)
		assert.NoError(t, err)
		assert.NotNil(t, product)
		assert.Equal(t, dto.Name, product.Name)
		if assert.NotNil(t, product.IDUser) {
			assert.Equal(t, userID, *product.IDUser)
		}
	})
}

func TestProductService_Update(t *testing.T) {
	repo := NewProductRepository(database.DB)
	service := NewProductService(repo)
	ctx := context.Background()

	p, _ := service.Create(ctx, CreateProductDTO{Name: "P", SKU: "SKU-UP", Category: "C", Price: 1})
	p2, _ := service.Create(ctx, CreateProductDTO{Name: "P2", SKU: "SKU-UP-2", Category: "C", Price: 1})

	t.Run("Success", func(t *testing.T) {
		res, err := service.Update(ctx, p.ID, map[string]interface{}{"name": "N"})
		assert.NoError(t, err)
		assert.Equal(t, "N", res.Name)
	})

	t.Run("Error - Duplicate SKU", func(t *testing.T) {
		_, err := service.Update(ctx, p.ID, map[string]interface{}{"sku": p2.SKU})
		assert.Error(t, err)
	})
}

func TestProductService_List(t *testing.T) {
	repo := NewProductRepository(database.DB)
	service := NewProductService(repo)
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
