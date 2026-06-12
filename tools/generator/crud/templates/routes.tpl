package {{.LowerName}}

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"backend-go/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	repo := New{{.Name}}Repository(db)
	svc := New{{.Name}}Service(repo)
	h := New{{.Name}}Handler(svc)

	{{.LowerName}}Routes := rg.Group("/{{.LowerName}}")
	{
		{{.LowerName}}Routes.GET("/:id", middleware.CheckPermission("{{.LowerName}}", "view"), h.GetByID)
		{{.LowerName}}Routes.GET("", middleware.CheckPermission("{{.LowerName}}", "view"), h.List)
		{{.LowerName}}Routes.GET("/all", middleware.CheckPermission("{{.LowerName}}", "view"), h.ListAll)
		{{.LowerName}}Routes.POST("", middleware.CheckPermission("{{.LowerName}}", "create"), h.Create)
		{{.LowerName}}Routes.PUT("/:id", middleware.CheckPermission("{{.LowerName}}", "create"), h.Update)
		{{.LowerName}}Routes.DELETE("/:id", middleware.CheckPermission("{{.LowerName}}", "delete"), h.Delete)
		{{.LowerName}}Routes.PATCH("/:id/status", middleware.CheckPermission("{{.LowerName}}", "activate"), h.SetStatus)
	}
}
