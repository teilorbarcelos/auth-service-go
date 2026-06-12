package user

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"github.com/teilorbarcelos/auth-service-go/internal/infra/pdf"
	"github.com/teilorbarcelos/auth-service-go/internal/infra/session"
	"github.com/teilorbarcelos/auth-service-go/internal/middleware"
	"github.com/teilorbarcelos/auth-service-go/pkg/config"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, sm session.SessionStore) {
	repo := NewUserRepository(db)
	pdfProvider := pdf.NewRemotePdfProvider(config.AppConfig.PdfServiceUrl)
	svc := NewUserService(repo, sm, pdfProvider)
	h := NewUserHandler(svc)

	userRoutes := rg.Group("/user")
	{
		userRoutes.GET("/export/pdf", middleware.CheckPermission("user", "view"), h.ExportPdf)
		userRoutes.GET("/:id", middleware.CheckPermission("user", "view"), h.GetByID)
		userRoutes.GET("", middleware.CheckPermission("user", "view"), h.List)
		userRoutes.GET("/all", middleware.CheckPermission("user", "view"), h.ListAll)
		userRoutes.POST("", middleware.CheckPermission("user", "create"), h.Create)
		userRoutes.PUT("/:id", middleware.CheckPermission("user", "create"), h.Update)
		userRoutes.DELETE("/:id", middleware.CheckPermission("user", "delete"), h.Delete)
		userRoutes.PATCH("/:id/status", middleware.CheckPermission("user", "activate"), h.SetStatus)
	}
}
