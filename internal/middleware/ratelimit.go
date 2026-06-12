package middleware

import (
	"net/http"
	"strconv"
	"time"

	"backend-go/pkg/cache"
	"backend-go/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

var rateLimitScript = redis.NewScript(`
local key = KEYS[1]
local window_ms = tonumber(ARGV[1])
local current = redis.call('INCR', key)
if current == 1 then
    redis.call('PEXPIRE', key, window_ms)
end
return {current, redis.call('PTTL', key)}
`)

func getRateLimitKey(c *gin.Context) string {
	userID, exists := c.Get("userID")
	if exists {
		return "ratelimit:user:" + userID.(string)
	}
	return "ratelimit:ip:" + c.ClientIP()
}

func getRateLimitConfig() (time.Duration, int64) {
	windowStr := config.AppConfig.RateLimitWindow
	windowDuration, err := time.ParseDuration(windowStr)
	if err != nil {
		windowDuration = time.Minute
	}
	maxRequests := int64(config.AppConfig.RateLimitMax)
	return windowDuration, maxRequests
}

func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.AppConfig.Environment == "test" {
			c.Next()
			return
		}

		key := getRateLimitKey(c)
		windowDuration, maxRequests := getRateLimitConfig()
		windowMs := windowDuration.Milliseconds()

		ctx := c.Request.Context()

		result, err := rateLimitScript.Run(ctx, cache.RedisClient, []string{key}, windowMs).Slice()
		if err != nil {
			c.Next()
			return
		}

		currentCount := int64(result[0].(int64))
		resetMs := result[1].(int64)

		remaining := maxRequests - currentCount
		if remaining < 0 {
			remaining = 0
		}

		resetInSeconds := resetMs / 1000

		c.Header("X-RateLimit-Limit", strconv.FormatInt(maxRequests, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetInSeconds, 10))

		if currentCount > maxRequests {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too Many Requests",
				"message": "Você excedeu o limite de requisições. Tente novamente em breve.",
				"details": gin.H{
					"limit":     maxRequests,
					"remaining": 0,
					"reset_in":  resetInSeconds,
				},
			})
			return
		}

		c.Next()
	}
}
