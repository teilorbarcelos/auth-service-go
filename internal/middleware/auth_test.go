package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"backend-go/pkg/cache"
	"backend-go/pkg/config"
	"backend-go/pkg/security"
)

var ctxBg = context.Background()

func TestAuthenticate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config.LoadConfig()
	if cache.RedisClient == nil {
		cache.ConnectRedis()
	}

	r := gin.New()
	r.Use(Authenticate())
	r.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get("userID")
		c.JSON(http.StatusOK, gin.H{"userID": userID})
	})

	sessionVersion := 1

	// 1. Sem header
	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 2. Token Inválido (Formato)
	req, _ = http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "InvalidFormat token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 2.1 Token Inválido (JWT)
	req, _ = http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 3. Token Válido mas sem session version no Redis
	token, _ := security.GenerateToken("user-123", "user@test.com", "role-admin", []security.Permission{{Feature: "user", View: true}}, sessionVersion)
	req, _ = http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 4. Token Válido com session version diferente
	versionKey := fmt.Sprintf(middlewareSessionVerKey, "user-123")
	cache.RedisClient.Set(ctxBg, versionKey, sessionVersion+1, 0)
	defer cache.RedisClient.Del(ctxBg, versionKey)

	token2, _ := security.GenerateToken("user-123", "user@test.com", "role-admin", []security.Permission{{Feature: "user", View: true}}, sessionVersion)
	req, _ = http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token2)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 5. Token Válido com session version correta
	cache.RedisClient.Set(ctxBg, versionKey, sessionVersion, 0)

	req, _ = http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user-123")
}
