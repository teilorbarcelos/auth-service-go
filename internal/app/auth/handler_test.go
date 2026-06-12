package auth

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
	"github.com/teilorbarcelos/auth-service-go/internal/core/domainerr"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LoginResponse), args.Error(1)
}

func (m *MockAuthService) GetMe(ctx context.Context, email string) (*LoginResponse, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LoginResponse), args.Error(1)
}

func (m *MockAuthService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LoginResponse), args.Error(1)
}

func (m *MockAuthService) Logout(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func TestAuthHandler_Login(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		h := NewHandler(mockSvc)
		r := gin.Default()
		r.POST("/login", h.Login)

		res := &LoginResponse{Valid: true, Token: "token"}
		mockSvc.On("Login", mock.Anything, "test@test.com", "pass").Return(res, nil)

		body, _ := json.Marshal(LoginRequest{Email: "test@test.com", Password: "pass"})
		req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		h := NewHandler(mockSvc)
		r := gin.Default()
		r.POST("/login", h.Login)

		req, _ := http.NewRequest("POST", "/login", bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Account Disabled Error", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		h := NewHandler(mockSvc)
		r := gin.Default()
		r.POST("/login", h.Login)

		mockSvc.On("Login", mock.Anything, "test@test.com", "pass").Return(nil, domainerr.ErrAccountDisabled)

		body, _ := json.Marshal(LoginRequest{Email: "test@test.com", Password: "pass"})
		req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Auth Not Configured Error", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		h := NewHandler(mockSvc)
		r := gin.Default()
		r.POST("/login", h.Login)

		mockSvc.On("Login", mock.Anything, "test@test.com", "pass").Return(nil, domainerr.ErrAuthNotConfigured)

		body, _ := json.Marshal(LoginRequest{Email: "test@test.com", Password: "pass"})
		req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("Internal Error", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		h := NewHandler(mockSvc)
		r := gin.Default()
		r.POST("/login", h.Login)

		mockSvc.On("Login", mock.Anything, "test@test.com", "pass").Return(nil, errors.New("generic error"))

		body, _ := json.Marshal(LoginRequest{Email: "test@test.com", Password: "pass"})
		req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAuthHandler_Me(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		h := NewHandler(mockSvc)
		r := gin.Default()
		r.GET("/me", func(c *gin.Context) {
			c.Set("userEmail", "test@test.com")
			h.Me(c)
		})

		res := &LoginResponse{Valid: true}
		mockSvc.On("GetMe", mock.Anything, "test@test.com").Return(res, nil)

		req, _ := http.NewRequest("GET", "/me", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("No Email in Context", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		h := NewHandler(mockSvc)
		r := gin.Default()
		r.GET("/me", h.Me)

		req, _ := http.NewRequest("GET", "/me", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		h := NewHandler(mockSvc)
		r := gin.Default()
		r.GET("/me", func(c *gin.Context) {
			c.Set("userEmail", "test@test.com")
			h.Me(c)
		})

		mockSvc.On("GetMe", mock.Anything, "test@test.com").Return(nil, domainerr.ErrUserNotFound)

		req, _ := http.NewRequest("GET", "/me", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthHandler_Refresh(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		h := NewHandler(mockSvc)
		r := gin.Default()
		r.POST("/refresh", h.Refresh)

		res := &LoginResponse{Valid: true, Token: "new-token"}
		mockSvc.On("RefreshToken", mock.Anything, "old-refresh").Return(res, nil)

		body, _ := json.Marshal(RefreshRequest{RefreshToken: "old-refresh"})
		req, _ := http.NewRequest("POST", "/refresh", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		h := NewHandler(mockSvc)
		r := gin.Default()
		r.POST("/refresh", h.Refresh)

		req, _ := http.NewRequest("POST", "/refresh", bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		h := NewHandler(mockSvc)
		r := gin.Default()
		r.POST("/refresh", h.Refresh)

		mockSvc.On("RefreshToken", mock.Anything, "old-refresh").Return(nil, domainerr.ErrInvalidCredentials)

		body, _ := json.Marshal(RefreshRequest{RefreshToken: "old-refresh"})
		req, _ := http.NewRequest("POST", "/refresh", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		h := NewHandler(mockSvc)
		r := gin.Default()
		r.POST("/logout", func(c *gin.Context) {
			c.Set("userID", "user-123")
			h.Logout(c)
		})

		mockSvc.On("Logout", mock.Anything, "user-123").Return(nil)

		req, _ := http.NewRequest("POST", "/logout", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("No UserID in Context", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		h := NewHandler(mockSvc)
		r := gin.Default()
		r.POST("/logout", h.Logout)

		req, _ := http.NewRequest("POST", "/logout", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		h := NewHandler(mockSvc)
		r := gin.Default()
		r.POST("/logout", func(c *gin.Context) {
			c.Set("userID", "user-123")
			h.Logout(c)
		})

		mockSvc.On("Logout", mock.Anything, "user-123").Return(errors.New("service error"))

		req, _ := http.NewRequest("POST", "/logout", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAuthHandler_JWKS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockAuthService)
	h := NewHandler(mockSvc)
	r := gin.Default()
	r.GET("/.well-known/jwks.json", h.JWKS)

	req, _ := http.NewRequest("GET", "/.well-known/jwks.json", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
