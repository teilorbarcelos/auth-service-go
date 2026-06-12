package user

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"backend-go/internal/core/models"
	"backend-go/internal/infra/pdf"
	"backend-go/internal/infra/session"
	"backend-go/pkg/config"
	"backend-go/pkg/database"
	"backend-go/pkg/security"
)

type failReader struct{}

func (failReader) Read(_ []byte) (int, error) {
	return 0, errors.New("read error")
}

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) Update(id string, updates map[string]interface{}) error {
	args := m.Called(id, updates)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(id string, preloads ...string) (*models.User, error) {
	// Variadic arguments in mock require special handling if we want to match them exactly,
	// but usually we can just pass them to Called.
	args := m.Called(id, preloads)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(email string, preloads ...string) (*models.User, error) {
	args := m.Called(email, preloads)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) UpdatePassword(authID string, password string) error {
	args := m.Called(authID, password)
	return args.Error(0)
}

func (m *MockUserRepository) IncrementSessionVersion(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockUserRepository) SearchPaginated(params database.FilterParams, filterable map[string]database.FilterConfig, searchable []database.SearchConfig, preloads ...string) ([]models.User, int64, error) {
	args := m.Called(params, filterable, searchable, preloads)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) WithContext(ctx context.Context) UserRepositoryI {
	// Usually WithContext returns itself or a new mock, but for simplicity we return the same mock
	return m
}

type MockPdfProvider struct {
	mock.Mock
}

func (m *MockPdfProvider) GeneratePdf(request pdf.PdfRequestDTO) (io.ReadCloser, error) {
	args := m.Called(request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}


func TestUserService_Create(t *testing.T) {
	repo := NewUserRepository(database.DB)
	sessionMgr := session.NewSessionManager()
	service := NewUserService(repo, sessionMgr, nil)

	dto := CreateUserDTO{
		Name:     "Test User",
		Email:    "test-create@example.com",
		Password: "password123",
		IDRole:   "administrator",
	}

	ctx := context.Background()
	user, err := service.Create(ctx, dto)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, dto.Name, user.Name)
	assert.Equal(t, dto.Email, user.Email)
	assert.NotEmpty(t, user.ID)
	assert.NotNil(t, user.Auth)
}

func TestUserService_Update(t *testing.T) {
	repo := NewUserRepository(database.DB)
	sessionMgr := session.NewSessionManager()
	service := NewUserService(repo, sessionMgr, nil)
	ctx := context.Background()

	// 1. Setup a regular user
	user, err := service.Create(ctx, CreateUserDTO{
		Name:     "Old Name",
		Email:    "old-update-unique@email.com",
		Password: "password",
		IDRole:   "administrator",
	})
	assert.NoError(t, err)

	t.Run("Update name and role", func(t *testing.T) {
		active := true
		updated, err := service.Update(ctx, user.ID, UpdateUserDTO{
			Name:   "New Name",
			IDRole: "manager",
			Active: &active,
		})
		if assert.NoError(t, err) && assert.NotNil(t, updated) {
			assert.Equal(t, "New Name", updated.Name)
			assert.Equal(t, "manager", updated.IDRole)
		}
	})

	t.Run("Update password", func(t *testing.T) {
		_, err := service.Update(ctx, user.ID, UpdateUserDTO{
			Password: "new-password",
		})
		assert.NoError(t, err)
	})

	t.Run("Update email", func(t *testing.T) {
		updated, err := service.Update(ctx, user.ID, UpdateUserDTO{
			Email: "new-email-unique@email.com",
		})
		if assert.NoError(t, err) && assert.NotNil(t, updated) {
			assert.Equal(t, "new-email-unique@email.com", updated.Email)
		}
	})

	t.Run("Admin protections", func(t *testing.T) {
		// Ensure admin user exists
		adminEmail := config.AppConfig.FirstUserEmail
		// Try to find it first
		foundAdmin, err := repo.WithContext(ctx).FindByEmail(adminEmail, "Auth")
		var admin *models.User
		if err != nil {
			// Create it if not found
			newAdmin, createErr := service.Create(ctx, CreateUserDTO{
				Name:     "Admin",
				Email:    adminEmail,
				Password: "password",
				IDRole:   "administrator",
			})
			assert.NoError(t, createErr, "should be able to create admin user")
			admin = newAdmin
		} else {
			admin = foundAdmin
		}
		
		assert.NotEmpty(t, admin.ID, "admin user ID should not be empty")
		assert.Equal(t, adminEmail, admin.Email, "admin email should match config")

		// Try to deactivate admin
		active := false
		_, err = service.Update(ctx, admin.ID, UpdateUserDTO{
			Active: &active,
		})
		if assert.Error(t, err) {
			assert.Equal(t, "o usuário administrador inicial não pode ser desativado", err.Error())
		}

		// Try to change admin email
		_, err = service.Update(ctx, admin.ID, UpdateUserDTO{
			Email: "other@email.com",
		})
		if assert.Error(t, err) {
			assert.Equal(t, "o email do usuário administrador inicial não pode ser alterado", err.Error())
		}
		
		// Change admin name (should be allowed)
		updated, err := service.Update(ctx, admin.ID, UpdateUserDTO{
			Name: "Updated Admin",
		})
		assert.NoError(t, err)
		assert.Equal(t, "Updated Admin", updated.Name)
	})
}

func TestUserService_List(t *testing.T) {
	repo := NewUserRepository(database.DB)
	sessionMgr := session.NewSessionManager()
	service := NewUserService(repo, sessionMgr, nil)

	// Create a user first
	ctx := context.Background()
	_, err := service.Create(ctx, CreateUserDTO{
		Name:     "List Test",
		Email:    "list-unique-test@example.com",
		Password: "password123",
		IDRole:   "administrator",
	})
	assert.NoError(t, err)

	params := database.FilterParams{
		Pagination: database.Pagination{
			Page:  1,
			Limit: 10,
		},
		Filters: map[string]interface{}{},
	}

	users, total, err := service.List(ctx, params)
	if assert.NoError(t, err) {
		assert.True(t, total > 0)
		assert.NotEmpty(t, users)
	}
}

func TestUserService_Delete(t *testing.T) {
	repo := NewUserRepository(database.DB)
	sessionMgr := session.NewSessionManager()
	service := NewUserService(repo, sessionMgr, nil)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		user, err := service.Create(ctx, CreateUserDTO{
			Name:     "Delete Me",
			Email:    "delete-unique-service@me.com",
			Password: "password",
			IDRole:   "administrator",
		})
		if assert.NoError(t, err) && assert.NotNil(t, user) {
			err = service.Delete(ctx, user.ID)
			assert.NoError(t, err)
	
			// Verify it's gone
			_, err = service.GetByID(ctx, user.ID)
			assert.Error(t, err)
		}
	})

	t.Run("Admin protection", func(t *testing.T) {
		// Admin is created in Update test or already exists
		// We can just find it by email
		if u, err := repo.WithContext(ctx).FindByEmail(config.AppConfig.FirstUserEmail); err == nil {
			err = service.Delete(ctx, u.ID)
			assert.Error(t, err)
			assert.Equal(t, "o usuário administrador inicial não pode ser excluído", err.Error())
		}
	})
}

func TestUserService_SetStatus(t *testing.T) {
	repo := NewUserRepository(database.DB)
	sessionMgr := session.NewSessionManager()
	service := NewUserService(repo, sessionMgr, nil)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		user, err := service.Create(ctx, CreateUserDTO{
			Name:     "Status Test",
			Email:    "status-unique-service@email.com",
			Password: "password",
			IDRole:   "administrator",
		})
		if assert.NoError(t, err) && assert.NotNil(t, user) {
			err = service.SetStatus(ctx, user.ID, false)
			assert.NoError(t, err)
	
			updated, _ := service.GetByID(ctx, user.ID)
			if assert.NotNil(t, updated) {
				assert.False(t, updated.Active)
			}
		}
	})

	t.Run("Admin protection", func(t *testing.T) {
		if u, err := repo.WithContext(ctx).FindByEmail(config.AppConfig.FirstUserEmail); err == nil {
			err = service.SetStatus(ctx, u.ID, false)
			assert.Error(t, err)
			assert.Equal(t, "o usuário administrador inicial não pode ser desativado", err.Error())
			
			// Activating admin should be allowed (even if already active)
			err = service.SetStatus(ctx, u.ID, true)
			assert.NoError(t, err)
		}
	})
}

func TestUserService_ErrorPaths(t *testing.T) {
	sessionMgr := session.NewSessionManager()
	ctx := context.Background()

	t.Run("Create Error", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		service := NewUserService(mockRepo, sessionMgr, nil)
		mockRepo.On("Create", mock.Anything).Return(errors.New("db error")).Once()
		_, err := service.Create(ctx, CreateUserDTO{Password: "pass"})
		assert.Error(t, err)
	})

	t.Run("Update FindByID Error", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		service := NewUserService(mockRepo, sessionMgr, nil)
		mockRepo.On("FindByID", "1", mock.Anything).Return(nil, errors.New("not found")).Once()
		_, err := service.Update(ctx, "1", UpdateUserDTO{})
		assert.Error(t, err)
	})

	t.Run("Update Repo Error", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		service := NewUserService(mockRepo, sessionMgr, nil)
		user := &models.User{Email: "test@test.com"}
		mockRepo.On("FindByID", "1", mock.Anything).Return(user, nil).Once()
		mockRepo.On("Update", "1", mock.Anything).Return(errors.New("update error")).Once()
		_, err := service.Update(ctx, "1", UpdateUserDTO{Name: "New"})
		assert.Error(t, err)
	})

	t.Run("Update Password Error", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		service := NewUserService(mockRepo, sessionMgr, nil)
		idAuth := "auth-id"
		user := &models.User{Email: "test@test.com", IDAuth: &idAuth}
		mockRepo.On("FindByID", "1", mock.Anything).Return(user, nil).Once() // Only once because it fails early
		mockRepo.On("UpdatePassword", idAuth, mock.Anything).Return(errors.New("pass error")).Once()
		_, err := service.Update(ctx, "1", UpdateUserDTO{Password: "new-pass"})
		assert.Error(t, err)
	})

	t.Run("Create Hash Error", func(t *testing.T) {
		oldReader := security.CryptoReader
		security.CryptoReader = failReader{}
		defer func() { security.CryptoReader = oldReader }()

		mockRepo := new(MockUserRepository)
		service := NewUserService(mockRepo, sessionMgr, nil)
		_, err := service.Create(ctx, CreateUserDTO{
			Name:     "Test",
			Email:    "hash-error@test.com",
			Password: "valid-password",
			IDRole:   "administrator",
		})
		assert.Error(t, err)
	})

	t.Run("Delete FindByID Error", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		service := NewUserService(mockRepo, sessionMgr, nil)
		mockRepo.On("FindByID", "1", mock.Anything).Return(nil, errors.New("not found")).Once()
		err := service.Delete(ctx, "1")
		assert.Error(t, err)
	})

	t.Run("Delete Repo Error", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		service := NewUserService(mockRepo, sessionMgr, nil)
		user := &models.User{Email: "test@test.com"}
		mockRepo.On("FindByID", "1", mock.Anything).Return(user, nil).Once()
		mockRepo.On("Update", "1", mock.Anything).Return(nil).Once()
		mockRepo.On("Delete", "1").Return(errors.New("delete error")).Once()
		err := service.Delete(ctx, "1")
		assert.Error(t, err)
	})

	t.Run("Delete Anonymize/Update Error", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		service := NewUserService(mockRepo, sessionMgr, nil)
		user := &models.User{Email: "test@test.com"}
		mockRepo.On("FindByID", "1", mock.Anything).Return(user, nil).Once()
		mockRepo.On("Update", "1", mock.Anything).Return(errors.New("update error")).Once()
		err := service.Delete(ctx, "1")
		assert.Error(t, err)
		assert.Equal(t, "update error", err.Error())
	})

	t.Run("SetStatus FindByID Error", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		service := NewUserService(mockRepo, sessionMgr, nil)
		mockRepo.On("FindByID", "1", mock.Anything).Return(nil, errors.New("not found")).Once()
		err := service.SetStatus(ctx, "1", true)
		assert.Error(t, err)
	})

	t.Run("SetStatus Repo Error", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		service := NewUserService(mockRepo, sessionMgr, nil)
		user := &models.User{Email: "test@test.com"}
		mockRepo.On("FindByID", "1", mock.Anything).Return(user, nil).Once()
		mockRepo.On("Update", "1", mock.Anything).Return(errors.New("update error")).Once()
		err := service.SetStatus(ctx, "1", true)
		assert.Error(t, err)
	})

	t.Run("IncrementSessionVersion Error", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		service := NewUserService(mockRepo, sessionMgr, nil)
		user := &models.User{Email: "test@test.com"}
		mockRepo.On("FindByID", "1", mock.Anything).Return(user, nil).Once()
		mockRepo.On("Update", "1", mock.Anything).Return(nil).Once()
		mockRepo.On("IncrementSessionVersion", mock.Anything, "1").Return(0, errors.New("version err"))

		err := service.SetStatus(ctx, "1", true)
		assert.NoError(t, err)
	})

	t.Run("SetSessionVersion Success", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		service := NewUserService(mockRepo, sessionMgr, nil)
		user := &models.User{Email: "test@test.com"}
		mockRepo.On("FindByID", "1", mock.Anything).Return(user, nil).Once()
		mockRepo.On("Update", "1", mock.Anything).Return(nil).Once()
		mockRepo.On("IncrementSessionVersion", mock.Anything, "1").Return(42, nil)

		err := service.SetStatus(ctx, "1", true)
		assert.NoError(t, err)
	})
}

func TestUserService_ExportPdf(t *testing.T) {
	sessionMgr := session.NewSessionManager()
	ctx := context.Background()

	t.Run("Success path", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		mockPdf := new(MockPdfProvider)
		service := NewUserService(mockRepo, sessionMgr, mockPdf)

		params := database.FilterParams{}
		roleName := "AdminRole"
		phone := "123456789"
		users := []models.User{
			{
				Name:   "User One",
				Email:  "one@example.com",
				Phone:  &phone,
				Active: true,
				Role: &models.Role{
					Name: roleName,
				},
			},
			{
				Name:   "User Two",
				Email:  "two@example.com",
				Phone:  nil,
				Active: false,
				Role:   nil,
			},
		}

		mockRepo.On("SearchPaginated", params, mock.Anything, mock.Anything, []string{"Role"}).Return(users, int64(2), nil).Once()

		expectedStream := io.NopCloser(bytes.NewReader([]byte("%PDF-1.4 mock content")))
		mockPdf.On("GeneratePdf", mock.Anything).Return(expectedStream, nil).Once()

		stream, err := service.ExportPdf(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, stream)

		content, err := io.ReadAll(stream)
		assert.NoError(t, err)
		assert.Equal(t, "%PDF-1.4 mock content", string(content))
	})

	t.Run("Search error", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		mockPdf := new(MockPdfProvider)
		service := NewUserService(mockRepo, sessionMgr, mockPdf)

		params := database.FilterParams{}
		mockRepo.On("SearchPaginated", params, mock.Anything, mock.Anything, []string{"Role"}).Return([]models.User{}, int64(0), errors.New("search error")).Once()

		stream, err := service.ExportPdf(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, stream)
		assert.Equal(t, "search error", err.Error())
	})

	t.Run("PDF generate error", func(t *testing.T) {
		mockRepo := new(MockUserRepository)
		mockPdf := new(MockPdfProvider)
		service := NewUserService(mockRepo, sessionMgr, mockPdf)

		params := database.FilterParams{}
		mockRepo.On("SearchPaginated", params, mock.Anything, mock.Anything, []string{"Role"}).Return([]models.User{}, int64(0), nil).Once()
		mockPdf.On("GeneratePdf", mock.Anything).Return(nil, errors.New("pdf generation failed")).Once()

		stream, err := service.ExportPdf(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, stream)
		assert.Equal(t, "pdf generation failed", err.Error())
	})
}
