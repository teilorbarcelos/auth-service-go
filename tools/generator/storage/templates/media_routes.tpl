package media

import (
	"github.com/gin-gonic/gin"
	"backend-go/pkg/storage"
	"os"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	// Pega o driver da env ou default s3
	driver := os.Getenv("STORAGE_DRIVER")
	if driver == "" {
		driver = "s3"
	}
	bucket := os.Getenv("STORAGE_BUCKET")
	if bucket == "" {
		bucket = "default-bucket"
	}

	provider, _ := storage.NewStorageProvider(driver, bucket)
	service := NewMediaService(provider)
	handler := NewMediaHandler(service)

	media := rg.Group("/media")
	{
		media.POST("/upload", handler.Upload)
	}
}
