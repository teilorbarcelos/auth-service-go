package role

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/redis/go-redis/v9"
	"backend-go/internal/core/models"
	"backend-go/internal/infra/session"
	"backend-go/pkg/cache"
	"backend-go/pkg/database"
)

type MockRoleRepository struct {
	mock.Mock
}

func (m *MockRoleRepository) WithContext(ctx context.Context) RoleRepositoryI {
	args := m.Called(ctx)
	return args.Get(0).(RoleRepositoryI)
}

func (m *MockRoleRepository) Create(role *models.Role) error {
	args := m.Called(role)
	return args.Error(0)
}

func (m *MockRoleRepository) Delete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockRoleRepository) Update(id string, updates map[string]interface{}) error {
	args := m.Called(id, updates)
	return args.Error(0)
}

func (m *MockRoleRepository) CreateWithPermissions(role *models.Role, perms []models.RoleFeature) error {
	args := m.Called(role, perms)
	return args.Error(0)
}

func (m *MockRoleRepository) UpdateWithPermissions(id string, role *models.Role, perms []models.RoleFeature) error {
	args := m.Called(id, role, perms)
	return args.Error(0)
}

func (m *MockRoleRepository) FindByID(id string, preloads ...string) (*models.Role, error) {
	args := m.Called(id, preloads)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRoleRepository) SearchPaginated(params database.FilterParams, f map[string]database.FilterConfig, s []database.SearchConfig, p ...string) ([]models.Role, int64, error) {
	args := m.Called(params, f, s, p)
	return args.Get(0).([]models.Role), args.Get(1).(int64), args.Error(2)
}

func (m *MockRoleRepository) ListFeatures(ctx context.Context) ([]models.Feature, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Feature), args.Error(1)
}

func (m *MockRoleRepository) BulkIncrementSessionVersion(ctx context.Context, roleID string) error {
	args := m.Called(ctx, roleID)
	return args.Error(0)
}

func (m *MockRoleRepository) FindUserIDsByRole(ctx context.Context, roleID string) ([]string, error) {
	args := m.Called(ctx, roleID)
	return args.Get(0).([]string), args.Error(1)
}

func TestRoleService_Create(t *testing.T) {
	repo := NewRoleRepository(database.DB)
	sessionMgr := session.NewSessionManager()
	service := NewRoleService(repo, sessionMgr)
	ctx := context.Background()

	dto := CreateRoleDTO{
		Name:        "Service Role 1",
		Description: "Service Description",
	}

	role, err := service.Create(ctx, dto)
	if assert.NoError(t, err) {
		assert.NotNil(t, role)
	}
}

func TestRoleService_Update(t *testing.T) {
	repo := NewRoleRepository(database.DB)
	sessionMgr := session.NewSessionManager()
	service := NewRoleService(repo, sessionMgr)
	ctx := context.Background()

	r1, _ := service.Create(ctx, CreateRoleDTO{Name: "R1", Description: "D"})

	t.Run("Success", func(t *testing.T) {
		dto := CreateRoleDTO{Name: "Updated R1-unique", Description: "D"}
		res, err := service.Update(ctx, r1.ID, dto)
		if assert.NoError(t, err) && assert.NotNil(t, res) {
			assert.Equal(t, "Updated R1-unique", res.Name)
		}
	})

	t.Run("Error - Duplicate Permissions", func(t *testing.T) {
		dto := CreateRoleDTO{
			Name: "R1", 
			Permissions: []models.RoleFeature{
				{IDFeature: "F1"},
				{IDFeature: "F1"},
			},
		}
		_, err := service.Update(ctx, r1.ID, dto)
		assert.Error(t, err)
	})
}

func TestRoleService_List(t *testing.T) {
	repo := NewRoleRepository(database.DB)
	sessionMgr := session.NewSessionManager()
	service := NewRoleService(repo, sessionMgr)
	ctx := context.Background()

	params := database.FilterParams{
		Pagination: database.Pagination{
			Page:  1,
			Limit: 10,
		},
	}

	items, _, err := service.List(ctx, params)
	assert.NoError(t, err)
	assert.NotNil(t, items)
}

func TestRoleService_Delete(t *testing.T) {
	sessionMgr := session.NewSessionManager()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		repo := NewRoleRepository(database.DB)
		service := NewRoleService(repo, sessionMgr)
		role, _ := service.Create(ctx, CreateRoleDTO{Name: "To Delete Service", Description: "D"})
		err := service.Delete(ctx, role.ID)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		mockRepo := new(MockRoleRepository)
		service := NewRoleService(mockRepo, sessionMgr)
		mockRepo.On("WithContext", mock.Anything).Return(mockRepo)
		mockRepo.On("Delete", "1").Return(errors.New("err"))

		err := service.Delete(ctx, "1")
		assert.Error(t, err)
	})
}

func TestRoleService_SetStatus(t *testing.T) {
	sessionMgr := session.NewSessionManager()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		repo := NewRoleRepository(database.DB)
		service := NewRoleService(repo, sessionMgr)
		role, _ := service.Create(ctx, CreateRoleDTO{Name: "To Status Service", Description: "D"})
		err := service.SetStatus(ctx, role.ID, false)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		mockRepo := new(MockRoleRepository)
		service := NewRoleService(mockRepo, sessionMgr)
		mockRepo.On("WithContext", mock.Anything).Return(mockRepo)
		mockRepo.On("Update", "1", mock.Anything).Return(errors.New("err"))

		err := service.SetStatus(ctx, "1", false)
		assert.Error(t, err)
	})

	t.Run("BulkIncrementSessionVersion Error", func(t *testing.T) {
		mockRepo := new(MockRoleRepository)
		service := NewRoleService(mockRepo, sessionMgr)
		mockRepo.On("WithContext", mock.Anything).Return(mockRepo).Maybe()
		mockRepo.On("Update", "1", mock.Anything).Return(nil).Once()
		mockRepo.On("BulkIncrementSessionVersion", mock.Anything, "1").Return(errors.New("bulk err")).Once()

		err := service.SetStatus(ctx, "1", false)
		assert.NoError(t, err)
	})

	t.Run("FindUserIDsByRole Error", func(t *testing.T) {
		mockRepo := new(MockRoleRepository)
		service := NewRoleService(mockRepo, sessionMgr)
		mockRepo.On("WithContext", mock.Anything).Return(mockRepo).Maybe()
		mockRepo.On("Update", "1", mock.Anything).Return(nil).Once()
		mockRepo.On("BulkIncrementSessionVersion", mock.Anything, "1").Return(nil).Once()
		mockRepo.On("FindUserIDsByRole", mock.Anything, "1").Return([]string{}, errors.New("find err")).Once()

		err := service.SetStatus(ctx, "1", false)
		assert.NoError(t, err)
	})

	t.Run("InvalidateUserSessions Error", func(t *testing.T) {
		oldClient := cache.RedisClient
		cache.RedisClient = redis.NewClient(&redis.Options{Addr: "invalid:6379"})
		defer func() { cache.RedisClient = oldClient }()

		mockRepo := new(MockRoleRepository)
		service := NewRoleService(mockRepo, sessionMgr)
		mockRepo.On("WithContext", mock.Anything).Return(mockRepo).Maybe()
		mockRepo.On("Update", "1", mock.Anything).Return(nil).Once()
		mockRepo.On("BulkIncrementSessionVersion", mock.Anything, "1").Return(nil).Once()
		mockRepo.On("FindUserIDsByRole", mock.Anything, "1").Return([]string{"redis-down-user"}, nil).Once()

		err := service.SetStatus(ctx, "1", false)
		assert.NoError(t, err)
	})
}
