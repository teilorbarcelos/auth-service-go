package middleware

import (
	"net/http"

	"backend-go/pkg/security"

	"github.com/gin-gonic/gin"
)

func CheckPermission(feature string, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleID, roleExists := c.Get("userRoleID")
		if roleExists && roleID.(string) == "administrator" {
			c.Next()
			return
		}

		bitset, exists := c.Get("userPermissionsBitset")
		if !exists || bitset == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permissões não encontradas no contexto. Faça login novamente."})
			c.Abort()
			return
		}

		if !bitset.(security.PermissionBitset).HasPermission(feature, action) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Você não tem permissão para realizar esta ação",
				"details": gin.H{
					"feature": feature,
					"action":  action,
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
