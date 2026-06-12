package dashboard

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"backend-go/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	repo := NewDashboardRepository(db)
	svc := NewDashboardService(repo)
	h := NewDashboardHandler(svc)

	dashboardRoutes := rg.Group("/dashboard")
	{
		dashboardRoutes.GET("/stats", middleware.CheckPermission("dashboard", "view"), h.GetStats)
	}
}
