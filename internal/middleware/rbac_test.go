package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"backend-go/pkg/security"
)

func TestCheckPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPermissions := []security.Permission{
		{Feature: "user", View: true, Create: false},
		{Feature: "product", View: true, Create: true, Delete: true},
	}
	mockBitset := security.CompilePermissions(mockPermissions)

	setupRouter := func(feature, action string) *gin.Engine {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			if c.Request.Header.Get("X-Role") != "" {
				c.Set("userRoleID", c.Request.Header.Get("X-Role"))
			}
			if c.Request.Header.Get("X-No-Perms") != "true" {
				c.Set("userPermissions", mockPermissions)
				c.Set("userPermissionsBitset", mockBitset)
			}
			c.Next()
		})
		r.GET("/test", CheckPermission(feature, action), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		return r
	}

	// 1. Bypass Administrador
	r := setupRouter("user", "create")
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Role", "administrator")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 2. Permissão Permitida (View User)
	r = setupRouter("user", "view")
	req, _ = http.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 3. Permissão Negada (Create User)
	r = setupRouter("user", "create")
	req, _ = http.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// 3.1 Permissão Permitida (Delete Product)
	r = setupRouter("product", "delete")
	req, _ = http.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 3.2 Permissão Negada (Activate Product - não definida no mock)
	r = setupRouter("product", "activate")
	req, _ = http.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// 3.3 Ação Inválida
	r = setupRouter("product", "invalid_action")
	req, _ = http.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// 4. Sem permissões no contexto
	r = setupRouter("user", "view")
	req, _ = http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-No-Perms", "true")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// 5. PermissõesBitset explicitamente nil
	r = gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userPermissionsBitset", nil)
		c.Next()
	})
	r.GET("/test", CheckPermission("user", "view"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	req, _ = http.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}
