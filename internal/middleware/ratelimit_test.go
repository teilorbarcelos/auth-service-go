package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend-go/pkg/cache"
	"backend-go/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config.LoadConfig()
	if cache.RedisClient == nil {
		cache.ConnectRedis()
	}

	r := gin.New()

	origEnv := config.AppConfig.Environment
	origMax := config.AppConfig.RateLimitMax
	origWindow := config.AppConfig.RateLimitWindow

	config.AppConfig.Environment = "development"
	config.AppConfig.RateLimitMax = 2
	config.AppConfig.RateLimitWindow = "10s"

	defer func() {
		config.AppConfig.Environment = origEnv
		config.AppConfig.RateLimitMax = origMax
		config.AppConfig.RateLimitWindow = origWindow
	}()

	r.Use(RateLimitMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	ctx := context.Background()
	key := "ratelimit:ip:127.0.0.1"
	cache.RedisClient.Del(ctx, key)

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "2", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "1", w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))

	t.Run("Redis Error Fail Open", func(t *testing.T) {
		oldClient := cache.RedisClient
		cache.RedisClient = redis.NewClient(&redis.Options{Addr: "localhost:1"})
		defer func() { cache.RedisClient = oldClient }()

		wErr := httptest.NewRecorder()
		r.ServeHTTP(wErr, req)
		assert.Equal(t, http.StatusOK, wErr.Code)
	})

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
	t.Run("Invalid Duration", func(t *testing.T) {
		cache.RedisClient.Del(ctx, key)
		config.AppConfig.RateLimitWindow = "invalid"
		defer func() { config.AppConfig.RateLimitWindow = "10s" }()

		wDur := httptest.NewRecorder()
		r.ServeHTTP(wDur, req)
		assert.Equal(t, http.StatusOK, wDur.Code)
	})

	t.Run("UserID Rate Limit", func(t *testing.T) {
		rUser := gin.New()
		rUser.Use(func(c *gin.Context) {
			c.Set("userID", "user-123")
			c.Next()
		})
		rUser.Use(RateLimitMiddleware())
		rUser.GET("/user", func(c *gin.Context) { c.Status(http.StatusOK) })

		cache.RedisClient.Del(ctx, "ratelimit:user:user-123")
		reqUser, _ := http.NewRequest("GET", "/user", nil)
		wUser := httptest.NewRecorder()
		rUser.ServeHTTP(wUser, reqUser)
		assert.Equal(t, http.StatusOK, wUser.Code)
		assert.Equal(t, "1", wUser.Header().Get("X-RateLimit-Remaining"))
	})

	t.Run("Bypass Environment Test", func(t *testing.T) {
		config.AppConfig.Environment = "test"
		defer func() { config.AppConfig.Environment = "development" }()

		wBypass := httptest.NewRecorder()
		r.ServeHTTP(wBypass, req)
		assert.Equal(t, http.StatusOK, wBypass.Code)
		assert.Empty(t, wBypass.Header().Get("X-RateLimit-Limit"))
	})

	t.Run("Negative TTL", func(t *testing.T) {
		cache.RedisClient.Set(ctx, key, 10, 0)
		defer cache.RedisClient.Del(ctx, key)

		wTTL := httptest.NewRecorder()
		r.ServeHTTP(wTTL, req)
		assert.Equal(t, http.StatusTooManyRequests, wTTL.Code)
	})
}
