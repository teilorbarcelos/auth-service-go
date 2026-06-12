package auth

import (
	"github.com/teilorbarcelos/auth-service-go/internal/infra/session"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(publicRG *gin.RouterGroup, protectedRG *gin.RouterGroup, db *gorm.DB) {
	sm := session.NewSessionManager()
	repo := NewRepository(db)
	svc := NewService(repo, sm)
	h := NewHandler(svc)

	authGroup := publicRG.Group("/auth")
	{
		authGroup.POST("/login", h.Login)
		authGroup.POST("/refresh", h.Refresh)
		authGroup.GET("/.well-known/jwks.json", h.JWKS)
	}
	protectedRG.POST("/auth/logout", h.Logout)
	protectedRG.GET("/auth/me", h.Me)
}
