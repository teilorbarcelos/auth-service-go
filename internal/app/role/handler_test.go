package role

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
	"backend-go/internal/infra/session"
	"backend-go/pkg/database"
)

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rg := r.Group("/api")
	sm := session.NewSessionManager()
	RegisterRoutes(rg, database.DB, sm)
	assert.NotNil(t, r)
}

type MockRoleService struct {
	mock.Mock
}

func (m *MockRoleService) ListFeatures(ctx context.Context) ([]models.Feature, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Feature), args.Error(1)
}

func (m *MockRoleService) Create(ctx context.Context, dto CreateRoleDTO) (*models.Role, error) {
	args := m.Called(ctx, dto)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRoleService) Update(ctx context.Context, id string, dto CreateRoleDTO) (*models.Role, error) {
	args := m.Called(ctx, id, dto)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRoleService) List(ctx context.Context, params database.FilterParams) ([]models.Role, int64, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]models.Role), args.Get(1).(int64), args.Error(2)
}

func (m *MockRoleService) GetByID(ctx context.Context, id string) (*models.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRoleService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRoleService) SetStatus(ctx context.Context, id string, active bool) error {
	args := m.Called(ctx, id, active)
	return args.Error(0)
}

func setupRoleHandler() (*RoleHandler, *gin.Engine) {
	repo := NewRoleRepository(database.DB)
	sessionMgr := session.NewSessionManager()
	service := NewRoleService(repo, sessionMgr)
	handler := NewRoleHandler(service)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	return handler, r
}

func setupMockHandler() (*RoleHandler, *gin.Engine, *MockRoleService) {
	mockService := new(MockRoleService)
	handler := NewRoleHandler(mockService)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	return handler, r, mockService
}

func TestRoleHandler_ListFeatures(t *testing.T) {
	h, r := setupRoleHandler()
	r.GET("/roles/features", h.ListFeatures)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/roles/features", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.GET("/roles/features", h.ListFeatures)
		mockSvc.On("ListFeatures", mock.Anything).Return([]models.Feature{}, errors.New("err"))

		req, _ := http.NewRequest(http.MethodGet, "/roles/features", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRoleHandler_Create(t *testing.T) {
	h, r := setupRoleHandler()
	r.POST("/roles", h.Create)

	t.Run("Success", func(t *testing.T) {
		dto := CreateRoleDTO{Name: "Role 1", Description: "Desc"}
		body, _ := json.Marshal(dto)
		req, _ := http.NewRequest(http.MethodPost, "/roles", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/roles", bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.POST("/roles", h.Create)
		mockSvc.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("err"))

		body, _ := json.Marshal(CreateRoleDTO{Name: "E", Description: "D"})
		req, _ := http.NewRequest(http.MethodPost, "/roles", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRoleHandler_Update(t *testing.T) {
	h, r := setupRoleHandler()
	r.PUT("/roles/:id", h.Update)

	repo := NewRoleRepository(database.DB)
	p := &models.Role{Name: "Old", Description: "Old"}
	repo.Create(p)

	t.Run("Success", func(t *testing.T) {
		body, _ := json.Marshal(CreateRoleDTO{Name: "New", Description: "New"})
		req, _ := http.NewRequest(http.MethodPut, "/roles/"+p.ID, bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, "/roles/1", bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.PUT("/roles/:id", h.Update)
		mockSvc.On("Update", mock.Anything, "1", mock.Anything).Return(nil, errors.New("err"))

		body, _ := json.Marshal(CreateRoleDTO{Name: "N", Description: "D"})
		req, _ := http.NewRequest(http.MethodPut, "/roles/1", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRoleHandler_GetByID(t *testing.T) {
	h, r := setupRoleHandler()
	r.GET("/roles/:id", h.GetByID)

	repo := NewRoleRepository(database.DB)
	p := &models.Role{Name: "Get", Description: "Get"}
	repo.Create(p)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/roles/"+p.ID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Not Found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/roles/non-existent", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestRoleHandler_List(t *testing.T) {
	h, r := setupRoleHandler()
	r.GET("/roles", h.List)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/roles?page=1&limit=10", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.GET("/roles", h.List)
		mockSvc.On("List", mock.Anything, mock.Anything).Return([]models.Role{}, int64(0), errors.New("err"))

		req, _ := http.NewRequest(http.MethodGet, "/roles", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRoleHandler_ListAll(t *testing.T) {
	h, r := setupRoleHandler()
	r.GET("/roles/all", h.ListAll)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/roles/all", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.GET("/roles/all", h.ListAll)
		mockSvc.On("List", mock.Anything, mock.Anything).Return([]models.Role{}, int64(0), errors.New("err"))

		req, _ := http.NewRequest(http.MethodGet, "/roles/all", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRoleHandler_Delete(t *testing.T) {
	h, r := setupRoleHandler()
	r.DELETE("/roles/:id", h.Delete)

	repo := NewRoleRepository(database.DB)
	p := &models.Role{Name: "Delete", Description: "Delete"}
	repo.Create(p)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, "/roles/"+p.ID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.DELETE("/roles/:id", h.Delete)
		mockSvc.On("Delete", mock.Anything, "1").Return(errors.New("err"))

		req, _ := http.NewRequest(http.MethodDelete, "/roles/1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRoleHandler_SetStatus(t *testing.T) {
	h, r := setupRoleHandler()
	r.PATCH("/roles/:id/status", h.SetStatus)

	repo := NewRoleRepository(database.DB)
	p := &models.Role{Name: "Status", Description: "Status"}
	repo.Create(p)

	t.Run("Success", func(t *testing.T) {
		body, _ := json.Marshal(map[string]bool{"active": false})
		req, _ := http.NewRequest(http.MethodPatch, "/roles/"+p.ID+"/status", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, "/roles/1/status", bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockSvc := setupMockHandler()
		r.PATCH("/roles/:id/status", h.SetStatus)
		mockSvc.On("SetStatus", mock.Anything, "1", false).Return(errors.New("err"))

		body, _ := json.Marshal(map[string]bool{"active": false})
		req, _ := http.NewRequest(http.MethodPatch, "/roles/1/status", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
