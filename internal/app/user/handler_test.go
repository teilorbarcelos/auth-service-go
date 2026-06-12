package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) Create(ctx context.Context, dto CreateUserDTO) (*models.User, error) {
	args := m.Called(ctx, dto)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) Update(ctx context.Context, id string, dto UpdateUserDTO) (*models.User, error) {
	args := m.Called(ctx, id, dto)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) List(ctx context.Context, params database.FilterParams) ([]models.User, int64, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserService) GetByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserService) SetStatus(ctx context.Context, id string, active bool) error {
	args := m.Called(ctx, id, active)
	return args.Error(0)
}

func (m *MockUserService) ExportPdf(ctx context.Context, params database.FilterParams) (io.ReadCloser, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func setupTestHandler() (*UserHandler, *gin.Engine) {
	repo := NewUserRepository(database.DB)
	sessionMgr := session.NewSessionManager()
	service := NewUserService(repo, sessionMgr, nil)
	handler := NewUserHandler(service)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	return handler, r
}

func setupMockHandler() (*UserHandler, *gin.Engine, *MockUserService) {
	mockService := new(MockUserService)
	handler := NewUserHandler(mockService)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	return handler, r, mockService
}

func createTestRole(t *testing.T) string {
	role := models.Role{
		Name:        "Test Role",
		Description: "Role for testing",
	}
	err := database.DB.Create(&role).Error
	assert.NoError(t, err)
	return role.ID
}

func TestNewUserHandler(t *testing.T) {
	repo := NewUserRepository(database.DB)
	sessionMgr := session.NewSessionManager()
	service := NewUserService(repo, sessionMgr, nil)
	h := NewUserHandler(service)

	assert.NotNil(t, h)
	assert.Equal(t, service, h.Service)
}

func TestUserHandler_Create(t *testing.T) {
	h, r := setupTestHandler()
	roleID := createTestRole(t)

	r.POST("/users", h.Create)

	t.Run("Success", func(t *testing.T) {
		dto := CreateUserDTO{
			Name:     "Handler Test",
			Email:    "handler@test.com",
			Password: "password123",
			IDRole:   roleID,
		}
		body, _ := json.Marshal(dto)
		req, _ := http.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		
		var res models.User
		err := json.Unmarshal(w.Body.Bytes(), &res)
		assert.NoError(t, err)
		assert.Equal(t, dto.Name, res.Name)
		assert.Equal(t, dto.Email, res.Email)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/users", bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockService := setupMockHandler()
		r.POST("/users", h.Create)

		mockService.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("service error"))

		dto := CreateUserDTO{
			Name:     "Error",
			Email:    "error@test.com",
			Password: "password123",
			IDRole:   roleID,
		}
		body, _ := json.Marshal(dto)
		req, _ := http.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUserHandler_Update(t *testing.T) {
	h, r := setupTestHandler()
	roleID := createTestRole(t)
	
	// Create a user to update
	user := models.User{
		Name:  "Update Me",
		Email: "update@me.com",
		IDRole: roleID,
	}
	database.DB.Create(&user)

	r.PUT("/users/:id", h.Update)

	t.Run("Success", func(t *testing.T) {
		newName := "Updated Name"
		dto := UpdateUserDTO{
			Name: newName,
		}
		body, _ := json.Marshal(dto)
		req, _ := http.NewRequest(http.MethodPut, "/users/"+user.ID, bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var res models.User
		json.Unmarshal(w.Body.Bytes(), &res)
		assert.Equal(t, newName, res.Name)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, "/users/"+user.ID, bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockService := setupMockHandler()
		r.PUT("/users/:id", h.Update)

		mockService.On("Update", mock.Anything, user.ID, mock.Anything).Return(nil, errors.New("service error"))

		dto := UpdateUserDTO{Name: "New Name"}
		body, _ := json.Marshal(dto)
		req, _ := http.NewRequest(http.MethodPut, "/users/"+user.ID, bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUserHandler_GetByID(t *testing.T) {
	h, r := setupTestHandler()
	roleID := createTestRole(t)
	
	user := models.User{
		Name:  "Get Me",
		Email: "get@me.com",
		IDRole: roleID,
	}
	database.DB.Create(&user)

	r.GET("/users/:id", h.GetByID)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/users/"+user.ID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var res models.User
		json.Unmarshal(w.Body.Bytes(), &res)
		assert.Equal(t, user.ID, res.ID)
	})

	t.Run("Not Found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/users/non-existent", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUserHandler_List(t *testing.T) {
	h, r := setupTestHandler()
	r.GET("/users", h.List)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/users?page=1&limit=10", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var res map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &res)
		assert.Contains(t, res, "items")
		assert.Contains(t, res, "total")
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockService := setupMockHandler()
		r.GET("/users", h.List)

		mockService.On("List", mock.Anything, mock.Anything).Return([]models.User{}, int64(0), errors.New("service error"))

		req, _ := http.NewRequest(http.MethodGet, "/users", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUserHandler_ListAll(t *testing.T) {
	h, r := setupTestHandler()
	r.GET("/users/all", h.ListAll)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/users/all", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var res map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &res)
		assert.Contains(t, res, "items")
		assert.Contains(t, res, "total")
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockService := setupMockHandler()
		r.GET("/users/all", h.ListAll)

		mockService.On("List", mock.Anything, mock.Anything).Return([]models.User{}, int64(0), errors.New("service error"))

		req, _ := http.NewRequest(http.MethodGet, "/users/all", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUserHandler_Delete(t *testing.T) {
	h, r := setupTestHandler()
	roleID := createTestRole(t)
	
	user := models.User{
		Name:  "Delete Me",
		Email: "delete@me.com",
		IDRole: roleID,
	}
	database.DB.Create(&user)

	r.DELETE("/users/:id", h.Delete)

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, "/users/"+user.ID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, "/users/non-existent-id", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUserHandler_SetStatus(t *testing.T) {
	h, r := setupTestHandler()
	roleID := createTestRole(t)
	
	user := models.User{
		Name:   "Status Test",
		Email:  "status@test.com",
		IDRole: roleID,
		Active: true,
	}
	database.DB.Create(&user)

	r.PATCH("/users/:id/status", h.SetStatus)

	t.Run("Success", func(t *testing.T) {
		body, _ := json.Marshal(map[string]bool{"active": false})
		req, _ := http.NewRequest(http.MethodPatch, "/users/"+user.ID+"/status", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, "/users/"+user.ID+"/status", bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockService := setupMockHandler()
		r.PATCH("/users/:id/status", h.SetStatus)

		mockService.On("SetStatus", mock.Anything, user.ID, false).Return(errors.New("service error"))

		body, _ := json.Marshal(map[string]bool{"active": false})
		req, _ := http.NewRequest(http.MethodPatch, "/users/"+user.ID+"/status", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUserHandler_ExportPdf(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		h, r, mockService := setupMockHandler()
		r.GET("/users/export/pdf", h.ExportPdf)

		pdfStream := io.NopCloser(bytes.NewReader([]byte("%PDF-1.4 mock content")))
		mockService.On("ExportPdf", mock.Anything, mock.Anything).Return(pdfStream, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/users/export/pdf", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
		assert.Equal(t, `attachment; filename="usuarios.pdf"`, w.Header().Get("Content-Disposition"))
		assert.Equal(t, "%PDF-1.4 mock content", w.Body.String())
	})

	t.Run("Service Error", func(t *testing.T) {
		h, r, mockService := setupMockHandler()
		r.GET("/users/export/pdf", h.ExportPdf)

		mockService.On("ExportPdf", mock.Anything, mock.Anything).Return(nil, errors.New("service error")).Once()

		req, _ := http.NewRequest(http.MethodGet, "/users/export/pdf", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Copy Error", func(t *testing.T) {
		h, r, mockService := setupMockHandler()
		r.GET("/users/export/pdf", h.ExportPdf)

		pdfStream := io.NopCloser(bytes.NewReader([]byte("%PDF-1.4 mock content")))
		mockService.On("ExportPdf", mock.Anything, mock.Anything).Return(pdfStream, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/users/export/pdf", nil)
		w := &errorResponseWriter{}
		r.ServeHTTP(w, req)
	})
}

type errorResponseWriter struct {
	header http.Header
}

func (e *errorResponseWriter) Header() http.Header {
	if e.header == nil {
		e.header = make(http.Header)
	}
	return e.header
}

func (e *errorResponseWriter) Write(b []byte) (int, error) {
	return 0, errors.New("write error")
}

func (e *errorResponseWriter) WriteHeader(statusCode int) {}
