package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"backend-go/pkg/cache"
		"backend-go/pkg/security"

	"github.com/gin-gonic/gin"
)

const middlewareSessionVerKey = "session:ver:%s"

func Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header é obrigatório"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Formato de autorização inválido. Use 'Bearer <token>'"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := security.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "UnauthorizedError"})
			c.Abort()
			return
		}

		storedVersion, err := cache.RedisClient.Get(c.Request.Context(), fmt.Sprintf(middlewareSessionVerKey, claims.UserID)).Int()
		if err != nil || storedVersion != claims.SessionVersion {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "UnauthorizedError"})
			c.Abort()
			return
		}

		ctx := context.WithValue(c.Request.Context(), "userID", claims.UserID)
		c.Request = c.Request.WithContext(ctx)

		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)
		c.Set("userRoleID", claims.RoleID)
		c.Set("userPermissions", claims.Permissions)
		if len(claims.Permissions) > 0 {
			c.Set("userPermissionsBitset", security.CompilePermissions(claims.Permissions))
		}

		c.Next()
	}
}
