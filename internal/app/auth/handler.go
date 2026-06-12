package auth

import (
	"errors"
	"net/http"

	"backend-go/internal/core/domainerr"

	"github.com/gin-gonic/gin"
)

const msgInvalidData = "dados inválidos"

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ValidateTokenRequest struct {
	Email string `json:"email" binding:"required,email"`
	Token string `json:"token" binding:"required"`
}

type ResetPasswordRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login realiza a autenticação do usuário
// @Summary Realizar Login
// @Description Autentica o usuário com email e senha, retornando tokens JWT e dados do usuário.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Credenciais de acesso"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} map[string]string "Dados inválidos"
// @Failure 401 {object} map[string]string "Credenciais inválidas"
// @Router /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": msgInvalidData})
		return
	}

	res, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// Me retorna os dados do usuário autenticado
// @Summary Obter dados do usuário atual
// @Description Retorna informações do usuário logado baseado no token JWT.
// @Tags Auth
// @Produce json
// @Security Bearer
// @Success 200 {object} LoginResponse
// @Failure 401 {object} map[string]string "Não autorizado"
// @Router /auth/me [get]
func (h *Handler) Me(c *gin.Context) {
	email, exists := c.Get("userEmail")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não identificado"})
		return
	}

	res, err := h.service.GetMe(c.Request.Context(), email.(string))
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// Refresh renova o token de acesso
// @Summary Renovar token
// @Description Gera um novo access token a partir de um refresh token válido.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} map[string]string "Dados inválidos"
// @Failure 401 {object} map[string]string "Refresh token inválido ou expirado"
// @Router /auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": msgInvalidData})
		return
	}

	res, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// ForgotPassword solicita um código de recuperação de senha
// @Summary Solicitar recuperação de senha
// @Description Envia um e-mail com um código de 6 dígitos para o usuário.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body ForgotPasswordRequest true "Email do usuário"
// @Success 200 {object} map[string]string "message: e-mail de recuperação enviado se o usuário existir"
// @Router /auth/password/request [post]
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email inválido"})
		return
	}

	h.service.RequestPasswordReset(c.Request.Context(), req.Email)

	c.JSON(http.StatusOK, gin.H{"message": "se o e-mail existir, um código de recuperação foi enviado"})
}

// ValidateToken valida o código de recuperação de senha
// @Summary Validar código de recuperação
// @Description Verifica se o código de 6 dígitos é válido e não expirou.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body ValidateTokenRequest true "Dados de validação"
// @Success 200 {object} map[string]bool "valid: true"
// @Failure 401 {object} map[string]string "Token inválido ou expirado"
// @Router /auth/password/validate [post]
func (h *Handler) ValidateToken(c *gin.Context) {
	var req ValidateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": msgInvalidData})
		return
	}

	valid, err := h.service.ValidateResetToken(c.Request.Context(), req.Email, req.Token)
	if err != nil || !valid {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

// ResetPassword define uma nova senha
// @Summary Redefinir senha
// @Description Altera a senha do usuário utilizando o código de validação.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body ResetPasswordRequest true "Novos dados de senha"
// @Success 200 {object} map[string]string "message: senha alterada com sucesso"
// @Failure 401 {object} map[string]string "Token inválido ou expirado"
// @Router /auth/password/change [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": msgInvalidData})
		return
	}

	err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Token, req.Password)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "senha alterada com sucesso"})
}

func (h *Handler) handleError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "erro interno do servidor"

	switch {
	case errors.Is(err, domainerr.ErrUserNotFound), errors.Is(err, domainerr.ErrInvalidCredentials), errors.Is(err, domainerr.ErrInvalidToken), errors.Is(err, domainerr.ErrTokenExpired), errors.Is(err, domainerr.ErrAccountDisabled):
		status = http.StatusUnauthorized
		message = "UnauthorizedError"
	case errors.Is(err, domainerr.ErrAuthNotConfigured):
		status = http.StatusUnprocessableEntity
		message = err.Error()
	}

	c.JSON(status, gin.H{"error": message})
}
