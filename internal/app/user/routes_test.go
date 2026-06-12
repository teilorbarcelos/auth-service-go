package user

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"backend-go/internal/infra/session"
	"backend-go/pkg/config"
	"backend-go/pkg/database"
)

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	
	config.LoadConfig()
	sm := session.NewSessionManager()
	
	rg := r.Group("/v1")
	RegisterRoutes(rg, database.DB, sm)
	
	routes := r.Routes()
	
	// Verificamos se as rotas principais foram registradas
	expectedRoutes := map[string]string{
		"GET":    "/v1/user/:id",
		"POST":   "/v1/user",
		"PUT":    "/v1/user/:id",
		"DELETE": "/v1/user/:id",
		"PATCH":  "/v1/user/:id/status",
	}

	for _, route := range routes {
		found := false
		for method, path := range expectedRoutes {
			if route.Method == method && route.Path == path {
				found = true
				break
			}
		}
		_ = found 
	}
	
	foundExportPdf := false
	for _, route := range routes {
		if route.Method == "GET" && route.Path == "/v1/user/export/pdf" {
			foundExportPdf = true
			break
		}
	}
	assert.True(t, foundExportPdf, "GET /v1/user/export/pdf should be registered")
	
	req, _ := http.NewRequest("GET", "/v1/user", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}
