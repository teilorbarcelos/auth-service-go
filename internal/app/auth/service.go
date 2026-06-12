package auth

import (
	"context"
	"fmt"
	"log"
	"time"

	"backend-go/internal/core/domainerr"
	"backend-go/internal/core/models"
	"backend-go/internal/infra/session"
	"backend-go/pkg/cache"
	"backend-go/pkg/email"
	"backend-go/pkg/security"
	"crypto/rand"
	"math/big"
)

var randReader = rand.Reader

type Service interface {
	Login(ctx context.Context, email, password string) (*LoginResponse, error)
	GetMe(ctx context.Context, email string) (*LoginResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error)
	RequestPasswordReset(ctx context.Context, email string) error
	ValidateResetToken(ctx context.Context, email, token string) (bool, error)
	ResetPassword(ctx context.Context, email, token, newPassword string) error
}

type authService struct {
	repo                 Repository
	sessionManager       session.SessionStore
	emailProvider        email.Provider
	GenerateToken        func(id, email, idRole string, permissions []security.Permission, sessionVersion int) (string, error)
	GenerateRefreshToken func(id, email, idRole string) (string, error)
	HashPassword         func(password string) (string, error)
}

func NewService(repo Repository, sessionMgr session.SessionStore) Service {
	return &authService{
		repo:                 repo,
		sessionManager:       sessionMgr,
		emailProvider:        email.NewMockEmailProvider(),
		GenerateToken:        security.GenerateToken,
		GenerateRefreshToken: security.GenerateRefreshToken,
		HashPassword:         security.HashPassword,
	}
}

type UserResponse struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Email     string      `json:"email"`
	Phone     *string     `json:"phone"`
	Document  *string     `json:"document"`
	Avatar    *string     `json:"avatar"`
	Active    bool        `json:"active"`
	IDRole    string      `json:"id_role"`
	CreatedAt string      `json:"created_at"`
	UpdatedAt string      `json:"updated_at"`
	Role      interface{} `json:"role"`
}

type LoginResponse struct {
	Message      string       `json:"message,omitempty"`
	Valid        bool         `json:"valid"`
	Token        string       `json:"token,omitempty"`
	RefreshToken string       `json:"refreshToken,omitempty"`
	User         UserResponse `json:"user"`
}

func (s *authService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, domainerr.ErrUserNotFound
	}

	if !user.Active || user.IsDeleted || (user.Role != nil && !user.Role.Active) {
		return nil, domainerr.ErrAccountDisabled
	}

	if user.Auth == nil || user.Auth.Password == nil {
		return nil, domainerr.ErrAuthNotConfigured
	}

	if !security.CheckPasswordHash(password, *user.Auth.Password) {
		return nil, domainerr.ErrInvalidCredentials
	}

	return s.prepareAuthResponse(ctx, user)
}

func (s *authService) GetMe(ctx context.Context, email string) (*LoginResponse, error) {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, domainerr.ErrUserNotFound
	}

	if !user.Active || user.IsDeleted || (user.Role != nil && !user.Role.Active) {
		return nil, domainerr.ErrAccountDisabled
	}

	res, err := s.prepareAuthResponse(ctx, user)
	if err != nil {
		return nil, err
	}
	res.Valid = true
	return res, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	claims, err := security.ValidateToken(refreshToken)
	if err != nil {
		return nil, domainerr.ErrInvalidCredentials
	}

	// Verificar se a sessão de refresh ainda existe no Redis
	tokenHash := security.SHA256(refreshToken)
	sessionKey := fmt.Sprintf("session:role:%s:user:%s:refresh:%s", claims.RoleID, claims.UserID, tokenHash)
	exists, err := cache.RedisClient.Exists(ctx, sessionKey).Result()
	if err != nil || exists == 0 {
		return nil, domainerr.ErrInvalidCredentials
	}

	user, err := s.repo.FindByEmail(ctx, claims.Email)
	if err != nil {
		return nil, domainerr.ErrUserNotFound
	}

	if !user.Active || user.IsDeleted || (user.Role != nil && !user.Role.Active) {
		return nil, domainerr.ErrAccountDisabled
	}

	return s.prepareAuthResponse(ctx, user)
}

func (s *authService) RequestPasswordReset(ctx context.Context, emailStr string) error {
	user, err := s.repo.FindByEmail(ctx, emailStr)
	if err != nil || user.Auth == nil {
		// Retornar sucesso silencioso por segurança para evitar enumeração de usuários
		return nil
	}

	token, _ := generateRandom6DigitToken()
	expiration := time.Now().Add(15 * time.Minute)

	updates := map[string]interface{}{
		"request_password_token":      token,
		"request_password_expiration": expiration,
	}

	if err := s.repo.UpdateAuth(ctx, user.Auth.ID, updates); err != nil {
		return err
	}

	return s.emailProvider.SendEmail(email.SendEmailParams{
		To:      emailStr,
		Subject: "Recuperação de Senha",
		Context: map[string]interface{}{
			"name":  user.Name,
			"token": token,
		},
	})
}

func (s *authService) ValidateResetToken(ctx context.Context, emailStr, token string) (bool, error) {
	user, err := s.repo.FindByEmail(ctx, emailStr)
	if err != nil {
		return false, domainerr.ErrUserNotFound
	}

	if user.Auth == nil || user.Auth.RequestPasswordToken == nil || *user.Auth.RequestPasswordToken != token {
		return false, domainerr.ErrInvalidToken
	}

	if user.Auth.RequestPasswordExpiration != nil && user.Auth.RequestPasswordExpiration.Before(time.Now()) {
		return false, domainerr.ErrTokenExpired
	}

	return true, nil
}

func (s *authService) ResetPassword(ctx context.Context, emailStr, token, newPassword string) error {
	user, err := s.repo.FindByEmail(ctx, emailStr)
	if err != nil {
		return domainerr.ErrUserNotFound
	}

	valid, err := s.ValidateResetToken(ctx, emailStr, token)
	if err != nil || !valid {
		return err
	}

	hashedPassword, err := s.HashPassword(newPassword)
	if err != nil {
		return domainerr.ErrInternal
	}

	updates := map[string]interface{}{
		"password":                    hashedPassword,
		"request_password_token":      nil,
		"request_password_expiration": nil,
		"retries":                     0,
	}

	return s.repo.UpdateAuth(ctx, user.Auth.ID, updates)
}

func generateRandom6DigitToken() (string, error) {
	n, err := rand.Int(randReader, big.NewInt(900000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}

func (s *authService) prepareAuthResponse(ctx context.Context, user *models.User) (*LoginResponse, error) {
	permissions := s.mapPermissions(user)

	sessionVersion := 0
	if user.Auth != nil {
		sessionVersion = user.Auth.SessionVersion
	}

	token, err := s.GenerateToken(user.ID, user.Email, user.IDRole, permissions, sessionVersion)
	if err != nil {
		return nil, domainerr.ErrInternal
	}

	refreshToken, err := s.GenerateRefreshToken(user.ID, user.Email, user.IDRole)
	if err != nil {
		return nil, domainerr.ErrInternal
	}

	s.sessionManager.SetSessionVersion(ctx, user.ID, sessionVersion)

	if err := s.createSession(ctx, user, token, refreshToken, permissions); err != nil {
		log.Printf("[AuthService] Erro ao salvar sessão: %v", err)
	}

	return &LoginResponse{
		Valid:        true,
		Token:        token,
		RefreshToken: refreshToken,
		User:         s.mapToUserResponse(user, permissions),
	}, nil
}

func (s *authService) mapPermissions(user *models.User) []security.Permission {
	permissions := make([]security.Permission, 0)
	for _, rf := range user.Role.RoleFeature {
		permissions = append(permissions, security.Permission{
			Feature:  rf.IDFeature,
			Create:   rf.Create,
			View:     rf.View,
			Delete:   rf.Delete,
			Activate: rf.Activate,
		})
	}
	return permissions
}

func (s *authService) createSession(ctx context.Context, user *models.User, token string, refreshToken string, permissions []security.Permission) error {
	payload := map[string]interface{}{
		"id":          user.ID,
		"email":       user.Email,
		"roleId":      user.IDRole,
		"permissions": permissions,
	}

	tokenHash := security.SHA256(token)
	refreshHash := security.SHA256(refreshToken)
	expireTime := 24 * time.Hour
	refreshExpireTime := 7 * 24 * time.Hour

	if err := s.sessionManager.CreateSession(ctx, user.ID, user.IDRole, tokenHash, payload, expireTime); err != nil {
		return err
	}
	return s.sessionManager.CreateRefreshToken(ctx, user.ID, user.IDRole, refreshHash, refreshExpireTime)
}

func (s *authService) mapToUserResponse(user *models.User, permissions []security.Permission) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Phone:     user.Phone,
		Document:  user.Document,
		Avatar:    user.Avatar,
		Active:    user.Active,
		IDRole:    user.IDRole,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
		Role: map[string]interface{}{
			"id":          user.Role.ID,
			"name":        user.Role.Name,
			"description": user.Role.Description,
			"active":      user.Role.Active,
			"created_at":  user.Role.CreatedAt.Format(time.RFC3339),
			"updated_at":  user.Role.UpdatedAt.Format(time.RFC3339),
			"permissions": permissions,
		},
	}
}
