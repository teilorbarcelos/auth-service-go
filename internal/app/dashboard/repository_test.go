package dashboard

import (
	"context"
	"testing"
	"time"

	"backend-go/pkg/database"
	"github.com/stretchr/testify/assert"
)

func TestDashboardRepository_Errors(t *testing.T) {
	repo := NewDashboardRepository(database.DB)

	// Create a canceled context to trigger repository errors
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	end := time.Now()

	t.Run("GetUserStats Error", func(t *testing.T) {
		res, err := repo.GetUserStats(ctx, start, end)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("GetProductStats Error", func(t *testing.T) {
		res, err := repo.GetProductStats(ctx, start, end)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("GetProductsPerUser Error", func(t *testing.T) {
		res, err := repo.GetProductsPerUser(ctx, start, end)
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}
