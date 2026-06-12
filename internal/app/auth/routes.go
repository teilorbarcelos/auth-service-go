package auth

import (
	"backend-go/internal/infra/session"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(publicRG *gin.RouterGroup, protectedRG *gin.RouterGroup, db *gorm.DB) {
	repo := NewRepository(db)
	sm := session.NewSessionManager()
	svc := NewService(repo, sm)
	h := NewHandler(svc)

	authGroup := publicRG.Group("/auth")
	{
		authGroup.POST("/login", h.Login)
		authGroup.POST("/refresh", h.Refresh)
		authGroup.POST("/password/request", h.ForgotPassword)
		authGroup.POST("/password/validate", h.ValidateToken)
		authGroup.POST("/password/change", h.ResetPassword)
	}
	protectedRG.GET("/auth/me", h.Me)
}
