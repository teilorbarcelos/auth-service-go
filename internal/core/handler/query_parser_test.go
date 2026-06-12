package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"backend-go/pkg/config"
)

func TestParseFilterParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Default values", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)

		params := ParseFilterParams(c)

		assert.Equal(t, 1, params.Pagination.Page)
		assert.Equal(t, 25, params.Pagination.Limit)
		assert.Equal(t, "", params.Order.OrderBy)
		assert.Equal(t, "", params.Order.OrderDirection)
		assert.Empty(t, params.Filters)
	})

	t.Run("Custom pagination and sorting", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/?page=2&limit=50&orderBy=name&orderDirection=desc", nil)

		params := ParseFilterParams(c)

		assert.Equal(t, 2, params.Pagination.Page)
		assert.Equal(t, 50, params.Pagination.Limit)
		assert.Equal(t, "name", params.Order.OrderBy)
		assert.Equal(t, "desc", params.Order.OrderDirection)
	})

	t.Run("Size parameter support (Frontend)", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/?size=100", nil)

		params := ParseFilterParams(c)

		assert.Equal(t, 100, params.Pagination.Limit)
	})

	t.Run("Search and filters", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/?searchWord=test&searchFields=name,email&isActive=true&isDeleted=false&role=admin", nil)

		params := ParseFilterParams(c)

		assert.Equal(t, "test", params.SearchWord)
		assert.Equal(t, "name,email", params.SearchFields)
		
		assert.Len(t, params.Filters, 3)
		assert.Equal(t, true, params.Filters["isActive"])
		assert.Equal(t, false, params.Filters["isDeleted"])
		assert.Equal(t, "admin", params.Filters["role"])
	})

	t.Run("Reserved keys should not be in filters", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/?page=1&limit=10&orderBy=id&custom=value", nil)

		params := ParseFilterParams(c)

		assert.Len(t, params.Filters, 1)
		assert.Equal(t, "value", params.Filters["custom"])
		assert.NotContains(t, params.Filters, "page")
		assert.NotContains(t, params.Filters, "limit")
		assert.NotContains(t, params.Filters, "orderBy")
	})

	t.Run("camelCase date keys normalized to snake_case", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/?createdAt=2026-05-14&updatedAt=2026-06-01&createdAt_start=2026-01-01&updatedAt_end=2026-12-31", nil)

		params := ParseFilterParams(c)

		assert.Equal(t, "2026-05-14", params.Filters["created_at"])
		assert.Equal(t, "2026-06-01", params.Filters["updated_at"])
		assert.Equal(t, "2026-01-01", params.Filters["created_at_start"])
		assert.Equal(t, "2026-12-31", params.Filters["updated_at_end"])
		assert.NotContains(t, params.Filters, "createdAt")
		assert.NotContains(t, params.Filters, "updatedAt")
	})
}

func TestHandleError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("NotFound Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		HandleError(c, gorm.ErrRecordNotFound)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.JSONEq(t, `{"error":"recurso não encontrado"}`, w.Body.String())
	})

	t.Run("BadRequest Errors", func(t *testing.T) {
		errorsToTest := []string{
			"filtro não está disponível",
			"campo obrigatório",
			"operação não é permitida",
			"acesso não é permitido",
		}

		for _, msg := range errorsToTest {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			HandleError(c, errors.New(msg))
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.JSONEq(t, `{"error":"`+msg+`"}`, w.Body.String())
		}
	})

	t.Run("InternalServerError", func(t *testing.T) {
		origEnv := config.AppConfig.Environment
		config.AppConfig.Environment = "development"
		defer func() { config.AppConfig.Environment = origEnv }()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		HandleError(c, errors.New("algum erro generico de banco"))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.JSONEq(t, `{"error":"algum erro generico de banco"}`, w.Body.String())
	})

	t.Run("InternalServerError in production sanitizes message", func(t *testing.T) {
		origEnv := config.AppConfig.Environment
		config.AppConfig.Environment = "production"
		defer func() { config.AppConfig.Environment = origEnv }()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		HandleError(c, errors.New("internal: connection to postgres failed with fatal error code 12345"))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.JSONEq(t, `{"error":"erro interno do servidor"}`, w.Body.String())
	})
}
