package product

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"backend-go/internal/core/models"
	"backend-go/pkg/database"
)

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rg := r.Group("/api")
	RegisterRoutes(rg, database.DB)
	assert.NotNil(t, r)
}

type MockProductService struct {
	mock.Mock
}

func (m *MockProductService) Create(ctx context.Context, dto CreateProductDTO) (*models.Product, error) {
	args := m.Called(ctx, dto)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockProductService) Update(ctx context.Context, id string, updates map[string]interface{}) (*models.Product, error) {
	args := m.Called(ctx, id, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockProductService) List(ctx context.Context, params database.FilterParams) ([]models.Product, int64, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]models.Product), args.Get(1).(int64), args.Error(2)
}

func (m *MockProductService) GetByID(ctx context.Context, id string) (*models.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockProductService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProductService) SetStatus(ctx context.Context, id string, active bool) error {
	args := m.Called(ctx, id, active)
	return args.Error(0)
}

func setupProductHandler() (*ProductHandler, *gin.Engine) {
	repo := NewProductRepository(database.DB)
	service := NewProductService(repo)
	handler := NewProductHandler(service)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	return handler, r
}

func setupMockHandler() (*ProductHandler, *gin.Engine, *MockProductService) {
	mockService := new(MockProductService)
	handler := NewProductHandler(mockService)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	return handler, r, mockService
}

func TestProductHandler_Create(t *testing.T) {
	h, r := setupProductHandler()
	r.POST("/products", h.Create)

	t.Run("Success", func(t *testing.T) {
		dto := CreateProductDTO{
			Name:     "Handler Test",
			SKU:      "SKU-U-H1",
			Category: "T",
			Price:    10.0,
		}
		body, _ := json.Marshal(dto)
		req, _ := http.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/products", bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.POST("/products", h.Create)
		mockSvc.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("err"))

		body, _ := json.Marshal(CreateProductDTO{Name: "E", SKU: "S", Category: "C", Price: 1})
		req, _ := http.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestProductHandler_Update(t *testing.T) {
	h, r := setupProductHandler()
	r.PUT("/products/:id", h.Update)

	repo := NewProductRepository(database.DB)
	p := &models.Product{Name: "P", SKU: "SKU-UP-1", Category: "C", Price: 1}
	repo.Create(p)

	t.Run("Success", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{"name": "N"})
		req, _ := http.NewRequest(http.MethodPut, "/products/"+p.ID, bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, "/products/1", bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.PUT("/products/:id", h.Update)
		mockSvc.On("Update", mock.Anything, "1", mock.Anything).Return(nil, errors.New("err"))

		body, _ := json.Marshal(map[string]interface{}{"name": "N"})
		req, _ := http.NewRequest(http.MethodPut, "/products/1", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestProductHandler_GetByID(t *testing.T) {
	h, r := setupProductHandler()
	r.GET("/products/:id", h.GetByID)

	repo := NewProductRepository(database.DB)
	p := &models.Product{Name: "G", SKU: "SKU-G1", Category: "C", Price: 1}
	repo.Create(p)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/products/"+p.ID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Not Found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/products/non-existent", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestProductHandler_List(t *testing.T) {
	h, r := setupProductHandler()
	r.GET("/products", h.List)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/products?page=1&limit=10", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.GET("/products", h.List)
		mockSvc.On("List", mock.Anything, mock.Anything).Return([]models.Product{}, int64(0), errors.New("err"))

		req, _ := http.NewRequest(http.MethodGet, "/products", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestProductHandler_ListAll(t *testing.T) {
	h, r := setupProductHandler()
	r.GET("/products/all", h.ListAll)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/products/all", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.GET("/products/all", h.ListAll)
		mockSvc.On("List", mock.Anything, mock.Anything).Return([]models.Product{}, int64(0), errors.New("err"))

		req, _ := http.NewRequest(http.MethodGet, "/products/all", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestProductHandler_Delete(t *testing.T) {
	h, r := setupProductHandler()
	r.DELETE("/products/:id", h.Delete)

	repo := NewProductRepository(database.DB)
	p := &models.Product{Name: "D", SKU: "SKU-D1", Category: "C", Price: 1}
	repo.Create(p)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, "/products/"+p.ID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.DELETE("/products/:id", h.Delete)
		mockSvc.On("Delete", mock.Anything, "1").Return(errors.New("err"))

		req, _ := http.NewRequest(http.MethodDelete, "/products/1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestProductHandler_SetStatus(t *testing.T) {
	h, r := setupProductHandler()
	r.PATCH("/products/:id/status", h.SetStatus)

	repo := NewProductRepository(database.DB)
	p := &models.Product{Name: "S", SKU: "SKU-S1", Category: "C", Price: 1}
	repo.Create(p)

	t.Run("Success", func(t *testing.T) {
		body, _ := json.Marshal(map[string]bool{"active": false})
		req, _ := http.NewRequest(http.MethodPatch, "/products/"+p.ID+"/status", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, "/products/1/status", bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.PATCH("/products/:id/status", h.SetStatus)
		mockSvc.On("SetStatus", mock.Anything, "1", false).Return(errors.New("err"))

		body, _ := json.Marshal(map[string]bool{"active": false})
		req, _ := http.NewRequest(http.MethodPatch, "/products/1/status", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
