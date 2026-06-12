package auth

import (
	"github.com/teilorbarcelos/auth-service-go/internal/infra/session"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(publicRG *gin.RouterGroup, protectedRG *gin.RouterGroup, db *gorm.DB) {
	repo := NewRepository(db)
	svc := NewService(repo, session.NewSessionManager())
	h := NewHandler(svc)

	authGroup := publicRG.Group("/auth")
	{
		authGroup.POST("/login", h.Login)
		authGroup.POST("/refresh", h.Refresh)
	}
	protectedRG.GET("/auth/me", h.Me)
}
