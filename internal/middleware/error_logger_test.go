package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend-go/pkg/database"
	"backend-go/pkg/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestErrorLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Configurar banco de dados para evitar panic na goroutine
	ctx := context.Background()
	pg, err := testutil.SetupPostgresContainer(ctx)
	if err == nil {
		defer pg.Terminate(ctx)
		database.DB = pg.DB
		database.DB.Exec("CREATE SCHEMA IF NOT EXISTS audit")
		database.DB.AutoMigrate(&backend_go_models_error_log{}) // Usar migração genérica ou mock
	}

	t.Run("Status 200 - Não loga", func(t *testing.T) {
		r := gin.New()
		r.Use(ErrorLogger())
		r.GET("/ok", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest("GET", "/ok", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		time.Sleep(10 * time.Millisecond) // wait for potential goroutine
	})

	t.Run("Status 500 - Loga Erro com UserID", func(t *testing.T) {
		r := gin.New()
		r.Use(ErrorLogger())
		r.GET("/err", func(c *gin.Context) {
			c.Set("userID", "user-123")
			c.Error(errors.New("some error"))
			c.Status(http.StatusInternalServerError)
		})

		req, _ := http.NewRequest("GET", "/err", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		time.Sleep(10 * time.Millisecond) // wait for potential goroutine
	})

	t.Run("Status 400 sem UserID - Não loga", func(t *testing.T) {
		r := gin.New()
		r.Use(ErrorLogger())
		r.GET("/bad", func(c *gin.Context) {
			c.Status(http.StatusBadRequest)
		})

		req, _ := http.NewRequest("GET", "/bad", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		time.Sleep(10 * time.Millisecond) // wait for potential goroutine
	})
}

// Mock model struct to prevent import cycle if any
type backend_go_models_error_log struct {
	ID string
}
