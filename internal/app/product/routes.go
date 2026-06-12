package product

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"backend-go/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	repo := NewProductRepository(db)
	svc := NewProductService(repo)
	h := NewProductHandler(svc)

	productRoutes := rg.Group("/product")
	{
		productRoutes.GET("/:id", middleware.CheckPermission("product", "view"), h.GetByID)
		productRoutes.GET("", middleware.CheckPermission("product", "view"), h.List)
		productRoutes.GET("/all", middleware.CheckPermission("product", "view"), h.ListAll)
		productRoutes.POST("", middleware.CheckPermission("product", "create"), h.Create)
		productRoutes.PUT("/:id", middleware.CheckPermission("product", "create"), h.Update)
		productRoutes.DELETE("/:id", middleware.CheckPermission("product", "delete"), h.Delete)
		productRoutes.PATCH("/:id/status", middleware.CheckPermission("product", "activate"), h.SetStatus)
	}
}
