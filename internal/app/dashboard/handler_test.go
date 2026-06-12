package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"backend-go/pkg/database"
)

func TestDashboardHandler_GetStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	repo := NewDashboardRepository(database.DB)
	service := NewDashboardService(repo)
	h := NewDashboardHandler(service)

	r.GET("/v1/dashboard/stats", h.GetStats)

	t.Run("Default Params", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/v1/dashboard/stats", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp DashboardStatsResponseDto
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.NotNil(t, resp.UserCreationStats)
		assert.NotNil(t, resp.ProductCreationStats)
		assert.NotNil(t, resp.ProductsPerUser)
	})

	t.Run("Filter Date Params", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/v1/dashboard/stats?createdAt_start=2026-05-10&createdAt_end=2026-05-20", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid Date Range", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/v1/dashboard/stats?createdAt_start=2026-05-20&createdAt_end=2026-05-10", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var body map[string]string
		json.Unmarshal(w.Body.Bytes(), &body)
		assert.Equal(t, "A data de início deve ser anterior ou igual à data de fim", body["error"])
	})

	t.Run("Invalid Date Format Fallback", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/v1/dashboard/stats?createdAt_start=invalid&createdAt_end=invalid", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		mockSvc := new(MockDashboardService)
		h := NewDashboardHandler(mockSvc)
		mockSvc.On("GetStats", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("service connection error"))

		rMock := gin.New()
		rMock.GET("/v1/dashboard/stats", h.GetStats)

		req, _ := http.NewRequest(http.MethodGet, "/v1/dashboard/stats", nil)
		w := httptest.NewRecorder()
		rMock.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var body map[string]string
		json.Unmarshal(w.Body.Bytes(), &body)
		assert.Equal(t, "service connection error", body["error"])
	})
}

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rg := r.Group("/api")
	RegisterRoutes(rg, database.DB)
	assert.NotNil(t, r)
}

type MockDashboardService struct {
	mock.Mock
}

func (m *MockDashboardService) GetStats(ctx context.Context, start, end time.Time) (*DashboardStatsResponseDto, error) {
	args := m.Called(ctx, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DashboardStatsResponseDto), args.Error(1)
}
