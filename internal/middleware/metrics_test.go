package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Record Metrics", func(t *testing.T) {
		r := gin.New()
		r.Use(Metrics())
		r.GET("/test/:id", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		beforeCount := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "/test/:id", "200"))

		req, _ := http.NewRequest("GET", "/test/1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		afterCount := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "/test/:id", "200"))
		assert.Equal(t, beforeCount+1, afterCount)
	})

	t.Run("Unknown Path", func(t *testing.T) {
		r := gin.New()
		r.Use(Metrics())
		// No route defined for this path to trigger "unknown" path in middleware
		
		beforeCount := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "unknown", "404"))

		req, _ := http.NewRequest("GET", "/not-found", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		
		afterCount := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "unknown", "404"))
		assert.Equal(t, beforeCount+1, afterCount)
	})
}
