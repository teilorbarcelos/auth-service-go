package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/teilorbarcelos/auth-service-go/internal/app/auth"
	"github.com/teilorbarcelos/auth-service-go/internal/middleware"
	"github.com/teilorbarcelos/auth-service-go/pkg/cache"
	"github.com/teilorbarcelos/auth-service-go/pkg/config"
	"github.com/teilorbarcelos/auth-service-go/pkg/database"
	"github.com/teilorbarcelos/auth-service-go/pkg/logger"

	"github.com/gin-gonic/gin"
)

func validateProductionConfig() {
	if len(config.AppConfig.JWTSecret) < 32 {
		logger.Log.Sugar().Fatalf("JWT_SECRET deve ter no mínimo 32 caracteres em produção")
	}
	if config.AppConfig.RateLimitMax <= 0 {
		logger.Log.Sugar().Fatalf("RATE_LIMIT_MAX deve ser maior que 0 em produção")
	}
	if _, err := time.ParseDuration(config.AppConfig.RateLimitWindow); err != nil {
		logger.Log.Sugar().Fatalf("RATE_LIMIT_WINDOW inválido (%s): %v", config.AppConfig.RateLimitWindow, err)
	}
}

func main() {
	config.LoadConfig()
	logger.InitLogger(config.AppConfig.Environment)

	if config.AppConfig.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
		validateProductionConfig()
	}

	database.ConnectDB()
	cache.ConnectRedis()

	const maxBodySize = 10 << 20

	r := gin.New()
	if config.AppConfig.TrustedProxies != "" {
		r.SetTrustedProxies(strings.Split(config.AppConfig.TrustedProxies, ","))
	} else {
		r.SetTrustedProxies(nil)
	}
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodySize)
		c.Next()
	})
	r.Use(middleware.CORS())
	r.Use(middleware.RateLimitMiddleware())
	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Next()
	})
	r.Use(middleware.Logger())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":      "ok",
			"environment": config.AppConfig.Environment,
		})
	})

	v1 := r.Group("/v1")
	{
		protected := v1.Group("/")
		protected.Use(middleware.Authenticate())
		auth.RegisterRoutes(v1, protected, database.DB)
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "rota não encontrada"})
	})
	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "método não permitido"})
	})

	addr := config.AppConfig.Host + ":" + config.AppConfig.Port
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		logger.Log.Sugar().Infof("Iniciando servidor em http://%s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Sugar().Fatalf("Erro ao iniciar servidor: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Encerrando servidor...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Sugar().Fatalf("Forçar encerramento do servidor: %v", err)
	}

	logger.Info("Servidor finalizado com sucesso.")
}
