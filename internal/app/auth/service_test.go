package auth

import (
	"context"
	"errors"
	"os"
	"testing"
	"testing/iotest"
	"time"

	"backend-go/internal/core/domainerr"
	"backend-go/internal/core/models"
	"backend-go/internal/infra/session"
	"backend-go/pkg/cache"
	"backend-go/pkg/security"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAuthRepository struct {
	mock.Mock
}

func (m *MockAuthRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthRepository) UpdateAuth(ctx context.Context, authID string, updates map[string]interface{}) error {
	args := m.Called(ctx, authID, updates)
	return args.Error(0)
}

func TestAuthService_Login(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	sm := session.NewSessionManager()
	serviceInterface := NewService(mockRepo, sm)
	service := serviceInterface.(*authService)
	ctx := context.Background()

	password := "password123"
	hashedPassword, _ := security.HashPassword(password)

	user := &models.User{
		BaseModel: models.BaseModel{ID: "1"},
		Email:     "test@test.com",
		Active:    true,
		Auth: &models.Auth{
			Password: &hashedPassword,
		},
		Role: &models.Role{
			BaseModel:   models.BaseModel{ID: "admin"},
			Name:        "Admin",
			Active:      true,
			RoleFeature: []models.RoleFeature{
				{IDFeature: "f1", View: true},
			},
		},
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("FindByEmail", mock.Anything, "test@test.com").Return(user, nil).Once()
		res, err := service.Login(ctx, "test@test.com", password)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.True(t, res.Valid)
	})

	t.Run("User Not Found", func(t *testing.T) {
		mockRepo.On("FindByEmail", mock.Anything, "notfound@test.com").Return(nil, os.ErrNotExist).Once()
		res, err := service.Login(ctx, "notfound@test.com", password)
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, "usuário não encontrado", err.Error())
	})

	t.Run("Inactive User", func(t *testing.T) {
		inactiveUser := *user
		inactiveUser.Active = false
		mockRepo.On("FindByEmail", mock.Anything, "inactive@test.com").Return(&inactiveUser, nil).Once()
		_, err := service.Login(ctx, "inactive@test.com", password)
		assert.Error(t, err)
		assert.Equal(t, "conta desativada ou removida", err.Error())
	})

	t.Run("No Auth Configured", func(t *testing.T) {
		noAuthUser := *user
		noAuthUser.Auth = nil
		mockRepo.On("FindByEmail", mock.Anything, "noauth@test.com").Return(&noAuthUser, nil).Once()
		_, err := service.Login(ctx, "noauth@test.com", password)
		assert.Error(t, err)
		assert.Equal(t, "autenticação não configurada para este usuário", err.Error())
	})

	t.Run("Invalid Password", func(t *testing.T) {
		mockRepo.On("FindByEmail", mock.Anything, "test@test.com").Return(user, nil).Once()
		_, err := service.Login(ctx, "test@test.com", "wrong")
		assert.Error(t, err)
		assert.Equal(t, "credenciais inválidas", err.Error())
	})

	t.Run("Redis Error", func(t *testing.T) {
		oldClient := cache.RedisClient
		cache.RedisClient = redis.NewClient(&redis.Options{Addr: "localhost:1"})
		defer func() { cache.RedisClient = oldClient }()

		mockRepo.On("FindByEmail", mock.Anything, "test@test.com").Return(user, nil).Once()
		res, err := service.Login(ctx, "test@test.com", password)
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("Token Error", func(t *testing.T) {
		oldGen := service.GenerateToken
		service.GenerateToken = func(id, email, idRole string, perms []security.Permission, sessionVersion int) (string, error) {
			return "", errors.New("token err")
		}
		defer func() { service.GenerateToken = oldGen }()

		mockRepo.On("FindByEmail", mock.Anything, "test@test.com").Return(user, nil).Once()
		_, err := service.Login(ctx, "test@test.com", password)
		assert.Error(t, err)
	})

	t.Run("Refresh Token Error", func(t *testing.T) {
		oldGen := service.GenerateRefreshToken
		service.GenerateRefreshToken = func(id, email, idRole string) (string, error) {
			return "", errors.New("refresh token err")
		}
		defer func() { service.GenerateRefreshToken = oldGen }()

		mockRepo.On("FindByEmail", mock.Anything, "test@test.com").Return(user, nil).Once()
		_, err := service.Login(ctx, "test@test.com", password)
		assert.Error(t, err)
	})
}

func TestAuthService_GetMe(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	sm := session.NewSessionManager()
	service := NewService(mockRepo, sm)
	ctx := context.Background()

	user := &models.User{
		BaseModel: models.BaseModel{ID: "1"},
		Email:     "test@test.com",
		Active:    true,
		Role: &models.Role{
			BaseModel: models.BaseModel{ID: "admin"},
			Active:    true,
		},
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("FindByEmail", mock.Anything, "test@test.com").Return(user, nil).Once()
		res, err := service.GetMe(ctx, "test@test.com")
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("Not Found", func(t *testing.T) {
		mockRepo.On("FindByEmail", mock.Anything, "error@test.com").Return(nil, os.ErrNotExist).Once()
		_, err := service.GetMe(ctx, "error@test.com")
		assert.Error(t, err)
	})

	t.Run("Inactive", func(t *testing.T) {
		inactive := *user
		inactive.Active = false
		mockRepo.On("FindByEmail", mock.Anything, "inactive@test.com").Return(&inactive, nil).Once()
		_, err := service.GetMe(ctx, "inactive@test.com")
		assert.Error(t, err)
	})

	t.Run("Token Error", func(t *testing.T) {
		svc := service.(*authService)
		oldGen := svc.GenerateToken
		svc.GenerateToken = func(id, email, idRole string, perms []security.Permission, sessionVersion int) (string, error) {
			return "", errors.New("token err")
		}
		defer func() { svc.GenerateToken = oldGen }()

		mockRepo.On("FindByEmail", mock.Anything, "test@test.com").Return(user, nil).Once()
		_, err := service.GetMe(ctx, "test@test.com")
		assert.Error(t, err)
	})
}

func TestAuthService_RefreshToken(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	sm := session.NewSessionManager()
	serviceInterface := NewService(mockRepo, sm)
	service := serviceInterface.(*authService)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		user := &models.User{
			BaseModel: models.BaseModel{ID: "success-user"},
			Email:     "success@test.com",
			Active:    true,
			Role: &models.Role{
				BaseModel: models.BaseModel{ID: "admin"},
				Active:    true,
			},
		}
		token, _ := security.GenerateRefreshToken(user.ID, user.Email, "admin")
		tokenHash := security.SHA256(token)
		sm.CreateRefreshToken(ctx, user.ID, "admin", tokenHash, time.Hour)
		
		mockRepo.On("FindByEmail", mock.Anything, user.Email).Return(user, nil).Once()
		res, err := service.RefreshToken(ctx, token)
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("Invalid Token", func(t *testing.T) {
		_, err := service.RefreshToken(ctx, "invalid")
		assert.Error(t, err)
	})

	t.Run("Session Expired", func(t *testing.T) {
		token, _ := security.GenerateRefreshToken("expired-user", "expired@test.com", "admin")
		_, err := service.RefreshToken(ctx, token)
		assert.Error(t, err)
		assert.Equal(t, domainerr.ErrInvalidCredentials, err)
	})

	t.Run("User Not Found", func(t *testing.T) {
		token, _ := security.GenerateRefreshToken("notfound-user", "notfound@test.com", "admin")
		tokenHash := security.SHA256(token)
		sm.CreateRefreshToken(ctx, "notfound-user", "admin", tokenHash, time.Hour)

		mockRepo.On("FindByEmail", mock.Anything, "notfound@test.com").Return(nil, os.ErrNotExist).Once()
		_, err := service.RefreshToken(ctx, token)
		assert.Error(t, err)
	})

	t.Run("Inactive User", func(t *testing.T) {
		user := &models.User{
			BaseModel: models.BaseModel{ID: "inactive-user"},
			Email:     "inactive-refresh@test.com",
			Active:    false,
			Role: &models.Role{
				BaseModel: models.BaseModel{ID: "admin"},
				Active:    true,
			},
		}
		token, _ := security.GenerateRefreshToken(user.ID, user.Email, "admin")
		tokenHash := security.SHA256(token)
		sm.CreateRefreshToken(ctx, user.ID, "admin", tokenHash, time.Hour)

		mockRepo.On("FindByEmail", mock.Anything, user.Email).Return(user, nil).Once()
		_, err := service.RefreshToken(ctx, token)
		assert.Error(t, err)
	})

	t.Run("Redis Error", func(t *testing.T) {
		token, _ := security.GenerateRefreshToken("redis-error-user", "redis@test.com", "admin")
		
		oldClient := cache.RedisClient
		cache.RedisClient = redis.NewClient(&redis.Options{Addr: "localhost:1"})
		defer func() { cache.RedisClient = oldClient }()

		_, err := service.RefreshToken(ctx, token)
		assert.Error(t, err)
		assert.Equal(t, domainerr.ErrInvalidCredentials, err)
	})
}

func TestAuthService_PasswordRecovery(t *testing.T) {
	mockRepo := new(MockAuthRepository)
	sm := session.NewSessionManager()
	serviceInterface := NewService(mockRepo, sm)
	service := serviceInterface.(*authService)
	ctx := context.Background()

	emailStr := "test@test.com"
	user := &models.User{
		BaseModel: models.BaseModel{ID: "1"},
		Name:      "Test User",
		Email:     emailStr,
		Auth: &models.Auth{
			BaseModel: models.BaseModel{ID: "auth-1"},
		},
	}

	t.Run("RequestPasswordReset_Success", func(t *testing.T) {
		mockRepo.On("FindByEmail", mock.Anything, emailStr).Return(user, nil).Once()
		mockRepo.On("UpdateAuth", mock.Anything, "auth-1", mock.Anything).Return(nil).Once()

		err := service.RequestPasswordReset(ctx, emailStr)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("RequestPasswordReset_UserNotFound", func(t *testing.T) {
		mockRepo.On("FindByEmail", mock.Anything, "unknown@test.com").Return(nil, os.ErrNotExist).Once()

		err := service.RequestPasswordReset(ctx, "unknown@test.com")
		assert.NoError(t, err) // Should return nil for security reasons
	})

	t.Run("ValidateResetToken_Success", func(t *testing.T) {
		token := "123456"
		exp := time.Now().Add(time.Hour)
		userWithToken := *user
		userWithToken.Auth = &models.Auth{
			RequestPasswordToken:      &token,
			RequestPasswordExpiration: &exp,
		}

		mockRepo.On("FindByEmail", mock.Anything, emailStr).Return(&userWithToken, nil).Once()

		valid, err := service.ValidateResetToken(ctx, emailStr, token)
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("ValidateResetToken_InvalidToken", func(t *testing.T) {
		token := "123456"
		userWithToken := *user
		userWithToken.Auth = &models.Auth{
			RequestPasswordToken: &token,
		}

		mockRepo.On("FindByEmail", mock.Anything, emailStr).Return(&userWithToken, nil).Once()

		valid, err := service.ValidateResetToken(ctx, emailStr, "wrong")
		assert.Error(t, err)
		assert.False(t, valid)
		assert.Equal(t, domainerr.ErrInvalidToken, err)
	})

	t.Run("ValidateResetToken_Expired", func(t *testing.T) {
		token := "123456"
		exp := time.Now().Add(-time.Hour)
		userWithToken := *user
		userWithToken.Auth = &models.Auth{
			RequestPasswordToken:      &token,
			RequestPasswordExpiration: &exp,
		}

		mockRepo.On("FindByEmail", mock.Anything, emailStr).Return(&userWithToken, nil).Once()

		valid, err := service.ValidateResetToken(ctx, emailStr, token)
		assert.Error(t, err)
		assert.False(t, valid)
		assert.Equal(t, domainerr.ErrTokenExpired, err)
	})

	t.Run("ResetPassword_Success", func(t *testing.T) {
		token := "123456"
		exp := time.Now().Add(time.Hour)
		userWithToken := *user
		userWithToken.Auth = &models.Auth{
			BaseModel:                 models.BaseModel{ID: "auth-1"},
			RequestPasswordToken:      &token,
			RequestPasswordExpiration: &exp,
		}

		mockRepo.On("FindByEmail", mock.Anything, emailStr).Return(&userWithToken, nil).Twice()
		mockRepo.On("UpdateAuth", mock.Anything, "auth-1", mock.MatchedBy(func(updates map[string]interface{}) bool {
			_, hasPassword := updates["password"]
			return hasPassword
		})).Return(nil).Once()

		err := service.ResetPassword(ctx, emailStr, token, "newPassword")
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("RequestPasswordReset_NoAuth", func(t *testing.T) {
		userNoAuth := *user
		userNoAuth.Auth = nil
		mockRepo.On("FindByEmail", mock.Anything, emailStr).Return(&userNoAuth, nil).Once()

		err := service.RequestPasswordReset(ctx, emailStr)
		assert.NoError(t, err) // Agora é silencioso
	})

	t.Run("RequestPasswordReset_UpdateError", func(t *testing.T) {
		mockRepo.On("FindByEmail", mock.Anything, emailStr).Return(user, nil).Once()
		mockRepo.On("UpdateAuth", mock.Anything, "auth-1", mock.Anything).Return(errors.New("db error")).Once()

		err := service.RequestPasswordReset(ctx, emailStr)
		assert.Error(t, err)
	})

	t.Run("ValidateResetToken_NoAuth", func(t *testing.T) {
		userNoAuth := *user
		userNoAuth.Auth = nil
		mockRepo.On("FindByEmail", mock.Anything, emailStr).Return(&userNoAuth, nil).Once()

		_, err := service.ValidateResetToken(ctx, emailStr, "123")
		assert.Error(t, err)
	})

	t.Run("ResetPassword_UserNotFound", func(t *testing.T) {
		mockRepo.On("FindByEmail", mock.Anything, "unknown").Return(nil, domainerr.ErrUserNotFound).Once()
		err := service.ResetPassword(ctx, "unknown", "123", "pass")
		assert.Error(t, err)
	})

	t.Run("ResetPassword_UpdateError", func(t *testing.T) {
		token := "123456"
		exp := time.Now().Add(time.Hour)
		userWithToken := *user
		userWithToken.Auth = &models.Auth{
			BaseModel:                 models.BaseModel{ID: "auth-1"},
			RequestPasswordToken:      &token,
			RequestPasswordExpiration: &exp,
		}

		mockRepo.On("FindByEmail", mock.Anything, emailStr).Return(&userWithToken, nil).Twice()
		mockRepo.On("UpdateAuth", mock.Anything, "auth-1", mock.Anything).Return(errors.New("db error")).Once()

		err := service.ResetPassword(ctx, emailStr, token, "newPassword")
		assert.Error(t, err)
	})

	t.Run("generateRandom6DigitToken_Error", func(t *testing.T) {
		oldReader := randReader
		randReader = iotest.ErrReader(errors.New("rand error"))
		defer func() { randReader = oldReader }()

		_, err := generateRandom6DigitToken()
		assert.Error(t, err)
	})
	t.Run("ValidateResetToken_UserNotFound", func(t *testing.T) {
		mockRepo.On("FindByEmail", mock.Anything, "unknown@test.com").Return(nil, domainerr.ErrUserNotFound).Once()

		valid, err := service.ValidateResetToken(ctx, "unknown@test.com", "123")
		assert.Error(t, err)
		assert.False(t, valid)
		assert.Equal(t, domainerr.ErrUserNotFound, err)
	})

	t.Run("ResetPassword_ValidationError", func(t *testing.T) {
		mockRepo.On("FindByEmail", mock.Anything, emailStr).Return(user, nil).Once()
		// ValidateResetToken will be called and it will call FindByEmail again
		mockRepo.On("FindByEmail", mock.Anything, emailStr).Return(user, nil).Once()

		err := service.ResetPassword(ctx, emailStr, "wrong-token", "newPass")
		assert.Error(t, err)
		assert.Equal(t, domainerr.ErrInvalidToken, err)
	})

	t.Run("ResetPassword_HashError", func(t *testing.T) {
		token := "123456"
		exp := time.Now().Add(time.Hour)
		userWithToken := *user
		userWithToken.Auth = &models.Auth{
			BaseModel:                 models.BaseModel{ID: "auth-1"},
			RequestPasswordToken:      &token,
			RequestPasswordExpiration: &exp,
		}

		mockRepo.On("FindByEmail", mock.Anything, emailStr).Return(&userWithToken, nil).Twice()

		service.HashPassword = func(password string) (string, error) {
			return "", errors.New("hash error")
		}
		defer func() { service.HashPassword = security.HashPassword }()

		err := service.ResetPassword(ctx, emailStr, token, "pass")
		assert.Error(t, err)
		assert.Equal(t, domainerr.ErrInternal, err)
	})
}
