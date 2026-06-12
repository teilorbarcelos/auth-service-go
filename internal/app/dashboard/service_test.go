package dashboard

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend-go/internal/core/models"
	"backend-go/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDashboardService_GetStats(t *testing.T) {
	repo := NewDashboardRepository(database.DB)
	service := NewDashboardService(repo)
	ctx := context.Background()

	// Clear tables before running tests
	database.DB.Exec("TRUNCATE TABLE product CASCADE")
	database.DB.Exec("TRUNCATE TABLE \"user\" CASCADE")
	database.DB.Exec("TRUNCATE TABLE role CASCADE")

	// Create test role to satisfy FK constraint
	role := &models.Role{
		Name:        "Administrator",
		Description: "Admin role",
	}
	role.ID = "administrator"
	err := database.DB.Create(role).Error
	assert.NoError(t, err)

	// Create test users
	user1 := &models.User{
		Name:   "User One",
		Email:  "user1@example.com",
		IDRole: "administrator",
	}
	user1.ID = "user-1"
	err = database.DB.Create(user1).Error
	assert.NoError(t, err)

	user2 := &models.User{
		Name:   "User Two",
		Email:  "user2@example.com",
		IDRole: "administrator",
	}
	user2.ID = "user-2"
	err = database.DB.Create(user2).Error
	assert.NoError(t, err)

	// Create test products
	p1 := &models.Product{
		Name:     "P1",
		SKU:      "SKU-P1",
		Category: "Cat1",
		Price:    10.0,
		IDUser:   &user1.ID,
	}
	p1.ID = "prod-1"
	err = database.DB.Create(p1).Error
	assert.NoError(t, err)

	p2 := &models.Product{
		Name:     "P2",
		SKU:      "SKU-P2",
		Category: "Cat2",
		Price:    20.0,
		IDUser:   &user2.ID,
	}
	p2.ID = "prod-2"
	err = database.DB.Create(p2).Error
	assert.NoError(t, err)

	p3 := &models.Product{
		Name:     "P3",
		SKU:      "SKU-P3",
		Category: "Cat2",
		Price:    30.0,
		IDUser:   &user2.ID,
	}
	p3.ID = "prod-3"
	err = database.DB.Create(p3).Error
	assert.NoError(t, err)

	// Anonymous product
	p4 := &models.Product{
		Name:     "P4",
		SKU:      "SKU-P4",
		Category: "Cat3",
		Price:    40.0,
		IDUser:   nil,
	}
	p4.ID = "prod-4"
	err = database.DB.Create(p4).Error
	assert.NoError(t, err)

	// Fetch stats
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now().Add(24 * time.Hour)

	stats, err := service.GetStats(ctx, start, end)
	assert.NoError(t, err)
	assert.NotNil(t, stats)

	// Assert userCreationStats and productCreationStats are populated
	assert.NotEmpty(t, stats.UserCreationStats)
	assert.NotEmpty(t, stats.ProductCreationStats)

	// Assert productsPerUser counts and fallbacks
	assert.Len(t, stats.ProductsPerUser, 3) // user2 (2), user1 (1), anonymous (1)

	// Sort order: user2 count=2, then others count=1
	assert.Equal(t, 2, stats.ProductsPerUser[0].Count)
	assert.Equal(t, "User Two", stats.ProductsPerUser[0].UserName)
	assert.Equal(t, "user-2", *stats.ProductsPerUser[0].UserID)

	// Check if anonymous exists
	var foundAnon bool
	for _, u := range stats.ProductsPerUser {
		if u.UserID == nil {
			assert.Equal(t, "Anonymous", u.UserName)
			assert.Equal(t, 1, u.Count)
			foundAnon = true
		}
	}
	assert.True(t, foundAnon)
}

type MockDashboardRepository struct {
	mock.Mock
}

func (m *MockDashboardRepository) GetUserStats(ctx context.Context, start, end time.Time) ([]TimeSeriesStatDto, error) {
	args := m.Called(ctx, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]TimeSeriesStatDto), args.Error(1)
}

func (m *MockDashboardRepository) GetProductStats(ctx context.Context, start, end time.Time) ([]TimeSeriesStatDto, error) {
	args := m.Called(ctx, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]TimeSeriesStatDto), args.Error(1)
}

func (m *MockDashboardRepository) GetProductsPerUser(ctx context.Context, start, end time.Time) ([]UserProductStatDto, error) {
	args := m.Called(ctx, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]UserProductStatDto), args.Error(1)
}

func TestDashboardService_GetStats_Errors(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	end := time.Now()
	dbErr := errors.New("database connection failed")

	t.Run("GetUserStats Error", func(t *testing.T) {
		mockRepo := new(MockDashboardRepository)
		service := NewDashboardService(mockRepo)

		mockRepo.On("GetUserStats", ctx, start, end).Return(nil, dbErr)

		stats, err := service.GetStats(ctx, start, end)
		assert.Error(t, err)
		assert.Nil(t, stats)
		assert.Equal(t, dbErr, err)
	})

	t.Run("GetProductStats Error", func(t *testing.T) {
		mockRepo := new(MockDashboardRepository)
		service := NewDashboardService(mockRepo)

		mockRepo.On("GetUserStats", ctx, start, end).Return([]TimeSeriesStatDto{}, nil)
		mockRepo.On("GetProductStats", ctx, start, end).Return(nil, dbErr)

		stats, err := service.GetStats(ctx, start, end)
		assert.Error(t, err)
		assert.Nil(t, stats)
		assert.Equal(t, dbErr, err)
	})

	t.Run("GetProductsPerUser Error", func(t *testing.T) {
		mockRepo := new(MockDashboardRepository)
		service := NewDashboardService(mockRepo)

		mockRepo.On("GetUserStats", ctx, start, end).Return([]TimeSeriesStatDto{}, nil)
		mockRepo.On("GetProductStats", ctx, start, end).Return([]TimeSeriesStatDto{}, nil)
		mockRepo.On("GetProductsPerUser", ctx, start, end).Return(nil, dbErr)

		stats, err := service.GetStats(ctx, start, end)
		assert.Error(t, err)
		assert.Nil(t, stats)
		assert.Equal(t, dbErr, err)
	})
}
