package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend-go/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		status        int
		method        string
		path          string
		requestID     string
		ginErrors     []error
		expectedLevel zapcore.Level
		expectedMsg   string
	}{
		{
			name:          "Success 200",
			status:        200,
			method:        "GET",
			path:          "/test",
			expectedLevel: zap.InfoLevel,
			expectedMsg:   "request",
		},
		{
			name:          "Client Error 400",
			status:        400,
			method:        "POST",
			path:          "/bad",
			expectedLevel: zap.WarnLevel,
			expectedMsg:   "client error",
		},
		{
			name:          "Server Error 500",
			status:        500,
			method:        "PUT",
			path:          "/error",
			expectedLevel: zap.ErrorLevel,
			expectedMsg:   "server error",
		},
		{
			name:          "Custom Request ID",
			status:        200,
			method:        "GET",
			path:          "/id",
			requestID:     "custom-id",
			expectedLevel: zap.InfoLevel,
			expectedMsg:   "request",
		},
		{
			name:          "Gin Errors",
			status:        200,
			method:        "GET",
			path:          "/gin-error",
			ginErrors:     []error{errors.New("test error 1"), errors.New("test error 2")},
			expectedLevel: zap.ErrorLevel,
			expectedMsg:   "test error 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observedLogger, logs := observer.New(zap.DebugLevel)
			oldLog := logger.Log
			logger.Log = zap.New(observedLogger)
			defer func() { logger.Log = oldLog }()

			r := gin.New()
			r.Use(Logger())
			
			handler := func(c *gin.Context) {
				for _, err := range tt.ginErrors {
					c.Error(err)
				}
				c.Status(tt.status)
			}

			r.GET(tt.path, handler)
			r.POST(tt.path, handler)
			r.PUT(tt.path, handler)

			req, _ := http.NewRequest(tt.method, tt.path, nil)
			if tt.requestID != "" {
				req.Header.Set("X-Request-ID", tt.requestID)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.status, w.Code)
			
			// Verify logs
			if len(tt.ginErrors) > 0 {
				// Manual filter for level since FilterLevel might be missing
				levelCount := 0
				for _, entry := range logs.All() {
					if entry.Level == tt.expectedLevel {
						levelCount++
					}
				}
				assert.Equal(t, len(tt.ginErrors), levelCount)
				
				for _, err := range tt.ginErrors {
					assert.Equal(t, 1, logs.FilterMessage(err.Error()).Len())
				}
			} else {
				// We filter by message because level might be common
				assert.Equal(t, 1, logs.FilterMessage(tt.expectedMsg).Len())
				assert.Equal(t, tt.expectedLevel, logs.FilterMessage(tt.expectedMsg).All()[0].Level)
			}

			// Check Request ID in response
			respID := w.Header().Get("X-Request-ID")
			assert.NotEmpty(t, respID)
			if tt.requestID != "" {
				assert.Equal(t, tt.requestID, respID)
			}
		})
	}
}
