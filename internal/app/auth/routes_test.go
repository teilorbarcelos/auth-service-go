package auth

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"backend-go/pkg/database"
)

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	
	publicRG := r.Group("/v1")
	protectedRG := r.Group("/v1")
	
	RegisterRoutes(publicRG, protectedRG, database.DB)
	
	routes := r.Routes()
	
	expectedRoutes := []struct {
		Method string
		Path   string
	}{
		{"POST", "/v1/auth/login"},
		{"POST", "/v1/auth/refresh"},
		{"GET", "/v1/auth/me"},
	}

	for _, expected := range expectedRoutes {
		found := false
		for _, route := range routes {
			if route.Method == expected.Method && route.Path == expected.Path {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected route %s %s not found", expected.Method, expected.Path)
	}
}
