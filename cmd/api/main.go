package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"backend-go/internal/app/auth"
	"backend-go/internal/app/dashboard"
	"backend-go/internal/app/product"
	"backend-go/internal/app/role"
	"backend-go/internal/app/user"
	"backend-go/internal/core/audit"
	"backend-go/internal/infra/session"
	"backend-go/internal/middleware"
	_ "backend-go/docs"
	"backend-go/pkg/cache"
	"backend-go/pkg/config"
	"backend-go/pkg/database"
	"backend-go/pkg/logger"
	"backend-go/pkg/messaging"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

var openAPISpec []byte

func loadOpenAPISpec() {
	data, err := os.ReadFile("./docs/swagger.json")
	if err != nil {
		logger.Warn("failed to read swagger.json", zap.Error(err))
		return
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(data, &spec); err != nil {
		logger.Warn("failed to parse swagger.json", zap.Error(err))
		openAPISpec = data
		return
	}

	if _, ok := spec["openapi"]; !ok {
		if v, ok := spec["swagger"]; ok {
			spec["openapi"] = v
		} else {
			spec["openapi"] = "3.0.0"
		}
	}

	openAPISpec, _ = json.Marshal(spec)
}

// @title Backend Go API
// @version 1.0
// @description API modular em Go com Gin e Swagger.
// @termsOfService http://swagger.io/terms/

// @contact.name Suporte API
// @contact.url http://www.swagger.io/support
// @contact.email suporte@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8888
// @BasePath /v1
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization

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
	audit.RegisterAuditHooks(database.DB)

	if config.AppConfig.Environment != "test" {
		auditBuffer := audit.NewAuditBuffer(database.DB, 50, 100*time.Millisecond)
		audit.SetAuditBuffer(auditBuffer)
		defer auditBuffer.Shutdown()
	}

	cache.ConnectRedis()
	messaging.ConnectRabbitMQ()

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
	r.Use(middleware.Metrics())
	r.Use(middleware.CORS())
	r.Use(middleware.RateLimitMiddleware())
	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Next()
	})
	r.Use(middleware.Logger())
	r.Use(middleware.ErrorLogger())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":      "ok",
			"environment": config.AppConfig.Environment,
		})
	})

	loadOpenAPISpec()

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/api-docs/openapi.json", func(c *gin.Context) {
		if openAPISpec != nil {
			c.Data(http.StatusOK, "application/json", openAPISpec)
		} else {
			c.File("./docs/swagger.json")
		}
	})

	sessionMgr := session.NewSessionManager()

	v1 := r.Group("/v1")
	{
			if config.AppConfig.Environment != "production" {
			v1.GET("/docs", func(c *gin.Context) {
				c.Redirect(http.StatusMovedPermanently, "/v1/docs/index.html")
			})
			v1.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/api-docs/openapi.json")))
		}
		protected := v1.Group("/")
		protected.Use(middleware.Authenticate())
		auth.RegisterRoutes(v1, protected, database.DB)
		user.RegisterRoutes(protected, database.DB, sessionMgr)
		role.RegisterRoutes(protected, database.DB, sessionMgr)
		product.RegisterRoutes(protected, database.DB)
		dashboard.RegisterRoutes(protected, database.DB)
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

	logger.Info("Limpando recursos...")
	if messaging.RabbitConn != nil {
		messaging.RabbitConn.Close()
	}

	logger.Info("Servidor finalizado com sucesso.")
}
