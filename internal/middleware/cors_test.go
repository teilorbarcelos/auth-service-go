package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Teste 1: Requisição normal
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))

	// Teste 2: Requisição OPTIONS (Preflight)
	req, _ = http.NewRequest("OPTIONS", "/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "POST, OPTIONS, GET, PUT, DELETE, PATCH", w.Header().Get("Access-Control-Allow-Methods"))
}
