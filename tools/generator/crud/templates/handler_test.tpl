package {{.LowerName}}

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

type Mock{{.Name}}Service struct {
	mock.Mock
}

func (m *Mock{{.Name}}Service) Create(ctx context.Context, dto Create{{.Name}}DTO) (*models.{{.Name}}, error) {
	args := m.Called(ctx, dto)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.{{.Name}}), args.Error(1)
}

func (m *Mock{{.Name}}Service) Update(ctx context.Context, id string, updates map[string]interface{}) (*models.{{.Name}}, error) {
	args := m.Called(ctx, id, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.{{.Name}}), args.Error(1)
}

func (m *Mock{{.Name}}Service) List(ctx context.Context, params database.FilterParams) ([]models.{{.Name}}, int64, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]models.{{.Name}}), args.Get(1).(int64), args.Error(2)
}

func (m *Mock{{.Name}}Service) GetByID(ctx context.Context, id string) (*models.{{.Name}}, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.{{.Name}}), args.Error(1)
}

func (m *Mock{{.Name}}Service) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *Mock{{.Name}}Service) SetStatus(ctx context.Context, id string, active bool) error {
	args := m.Called(ctx, id, active)
	return args.Error(0)
}

func setup{{.Name}}Handler() (*{{.Name}}Handler, *gin.Engine) {
	repo := New{{.Name}}Repository(database.DB)
	service := New{{.Name}}Service(repo)
	handler := New{{.Name}}Handler(service)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	return handler, r
}

func setupMockHandler() (*{{.Name}}Handler, *gin.Engine, *Mock{{.Name}}Service) {
	mockService := new(Mock{{.Name}}Service)
	handler := New{{.Name}}Handler(mockService)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	return handler, r, mockService
}

func Test{{.Name}}Handler_Create(t *testing.T) {
	h, r := setup{{.Name}}Handler()
	r.POST("/{{.LowerName}}", h.Create)

	t.Run("Success", func(t *testing.T) {
		dto := Create{{.Name}}DTO{Name: "Handler Test"}
		body, _ := json.Marshal(dto)
		req, _ := http.NewRequest(http.MethodPost, "/{{.LowerName}}", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/{{.LowerName}}", bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.POST("/{{.LowerName}}", h.Create)
		mockSvc.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("err"))

		body, _ := json.Marshal(Create{{.Name}}DTO{Name: "E"})
		req, _ := http.NewRequest(http.MethodPost, "/{{.LowerName}}", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func Test{{.Name}}Handler_Update(t *testing.T) {
	h, r := setup{{.Name}}Handler()
	r.PUT("/{{.LowerName}}/:id", h.Update)

	repo := New{{.Name}}Repository(database.DB)
	c := &models.{{.Name}}{Name: "C"}
	repo.Create(c)

	t.Run("Success", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{"name": "N"})
		req, _ := http.NewRequest(http.MethodPut, "/{{.LowerName}}/"+c.ID, bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, "/{{.LowerName}}/1", bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.PUT("/{{.LowerName}}/:id", h.Update)
		mockSvc.On("Update", mock.Anything, "1", mock.Anything).Return(nil, errors.New("err"))

		body, _ := json.Marshal(map[string]interface{}{"name": "N"})
		req, _ := http.NewRequest(http.MethodPut, "/{{.LowerName}}/1", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func Test{{.Name}}Handler_GetByID(t *testing.T) {
	h, r := setup{{.Name}}Handler()
	r.GET("/{{.LowerName}}/:id", h.GetByID)

	repo := New{{.Name}}Repository(database.DB)
	c := &models.{{.Name}}{Name: "G"}
	repo.Create(c)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/{{.LowerName}}/"+c.ID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Not Found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/{{.LowerName}}/non-existent", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func Test{{.Name}}Handler_List(t *testing.T) {
	h, r := setup{{.Name}}Handler()
	r.GET("/{{.LowerName}}", h.List)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/{{.LowerName}}?page=1&limit=10", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.GET("/{{.LowerName}}", h.List)
		mockSvc.On("List", mock.Anything, mock.Anything).Return([]models.{{.Name}}{}, int64(0), errors.New("err"))

		req, _ := http.NewRequest(http.MethodGet, "/{{.LowerName}}", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func Test{{.Name}}Handler_ListAll(t *testing.T) {
	h, r := setup{{.Name}}Handler()
	r.GET("/{{.LowerName}}/all", h.ListAll)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/{{.LowerName}}/all", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.GET("/{{.LowerName}}/all", h.ListAll)
		mockSvc.On("List", mock.Anything, mock.Anything).Return([]models.{{.Name}}{}, int64(0), errors.New("err"))

		req, _ := http.NewRequest(http.MethodGet, "/{{.LowerName}}/all", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func Test{{.Name}}Handler_Delete(t *testing.T) {
	h, r := setup{{.Name}}Handler()
	r.DELETE("/{{.LowerName}}/:id", h.Delete)

	repo := New{{.Name}}Repository(database.DB)
	c := &models.{{.Name}}{Name: "D"}
	repo.Create(c)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, "/{{.LowerName}}/"+c.ID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.DELETE("/{{.LowerName}}/:id", h.Delete)
		mockSvc.On("Delete", mock.Anything, "1").Return(errors.New("err"))

		req, _ := http.NewRequest(http.MethodDelete, "/{{.LowerName}}/1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func Test{{.Name}}Handler_SetStatus(t *testing.T) {
	h, r := setup{{.Name}}Handler()
	r.PATCH("/{{.LowerName}}/:id/status", h.SetStatus)

	repo := New{{.Name}}Repository(database.DB)
	c := &models.{{.Name}}{Name: "S"}
	repo.Create(c)

	t.Run("Success", func(t *testing.T) {
		body, _ := json.Marshal(map[string]bool{"active": false})
		req, _ := http.NewRequest(http.MethodPatch, "/{{.LowerName}}/"+c.ID+"/status", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, "/{{.LowerName}}/1/status", bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.PATCH("/{{.LowerName}}/:id/status", h.SetStatus)
		mockSvc.On("SetStatus", mock.Anything, "1", false).Return(errors.New("err"))

		body, _ := json.Marshal(map[string]bool{"active": false})
		req, _ := http.NewRequest(http.MethodPatch, "/{{.LowerName}}/1/status", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
