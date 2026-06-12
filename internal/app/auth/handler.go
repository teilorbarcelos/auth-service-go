package auth

import (
	"errors"
	"net/http"

	"github.com/teilorbarcelos/auth-service-go/internal/core/domainerr"

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

func (h *Handler) Logout(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não identificado"})
		return
	}

	err := h.service.Logout(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao revogar sessão"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": true})
}

func (h *Handler) JWKS(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"keys": []interface{}{}})
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
