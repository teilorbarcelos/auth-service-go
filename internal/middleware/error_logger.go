package middleware

import (
	"fmt"
	"time"

	"backend-go/internal/core/models"
	"backend-go/pkg/database"

	"github.com/gin-gonic/gin"
)

func ErrorLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		status := c.Writer.Status()
		if status >= 400 {
			var userID *string
			if u, exists := c.Get("userID"); exists {
				uidStr := u.(string)
				userID = &uidStr
			}

			// Skip error logging for unauthenticated requests
			if userID == nil {
				return
			}

			// Capture the error message from response or GORM errors
			errorMessage := fmt.Sprintf("HTTP Error %d", status)
			if len(c.Errors) > 0 {
				errorMessage = c.Errors.String()
			}

			errorLog := models.ErrorLog{
				IDUser:       userID,
				Source:       c.Request.Method + " " + c.Request.URL.Path,
				ErrorMessage: errorMessage,
				ErrorData:    errorMessage,
				CreatedAt:    time.Now(),
			}

			// Save to database asynchronously
			go func() {
				database.DB.Create(&errorLog)
			}()
		}
	}
}
