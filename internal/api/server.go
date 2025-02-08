package api

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pdf-service-go/internal/api/middleware"
	"pdf-service-go/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Server struct {
	Router   *gin.Engine
	Handlers *Handlers
	server   *http.Server
}

func NewServer(handlers *Handlers) *Server {
	// Отключаем стандартный логгер gin
	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()

	router := gin.New() // Используем gin.New() вместо gin.Default()

	// Настройка лимитов
	router.MaxMultipartMemory = 8 << 20 // 8 MiB
	logger.Info("Server memory limits configured", zap.String("max_multipart_memory", "8 MiB"))

	// Добавляем middleware для восстановления после паники
	router.Use(gin.Recovery())

	// Добавляем middleware для фильтрации health check логов
	router.Use(func(c *gin.Context) {
		// Пропускаем логирование для health check
		if c.Request.URL.Path != "/health" {
			// Логируем начало запроса
			start := time.Now()
			path := c.Request.URL.Path
			raw := c.Request.URL.RawQuery

			c.Next()

			// Логируем результат запроса
			latency := time.Since(start)
			clientIP := c.ClientIP()
			method := c.Request.Method
			statusCode := c.Writer.Status()

			if raw != "" {
				path = path + "?" + raw
			}

			logger.Info("HTTP Request",
				zap.String("client_ip", clientIP),
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status_code", statusCode),
				zap.Duration("latency", latency),
			)
		} else {
			c.Next()
		}
	})

	// Добавляем middleware для метрик
	router.Use(middleware.PrometheusMiddleware())

	// Добавляем middleware для таймаутов
	router.Use(func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	logger.Info("Server middleware configured")

	return &Server{
		Router:   router,
		Handlers: handlers,
	}
}

func (s *Server) SetupRoutes() {
	// Health check для k8s
	s.Router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Метрики Prometheus
	s.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API endpoints
	v1 := s.Router.Group("/api/v1")
	{
		v1.POST("/docx", s.Handlers.PDF.GenerateDocx)
	}

	// Поддержка старого endpoint'а для обратной совместимости
	s.Router.POST("/generate-pdf", s.Handlers.PDF.GenerateDocx)

	logger.Info("Routes configured",
		logger.Field("health_endpoint", "/health"),
		logger.Field("metrics_endpoint", "/metrics"),
		logger.Field("api_endpoints", []string{"/api/v1/docx", "/generate-pdf"}),
	)
}

func (s *Server) Start(addr string) error {
	s.server = &http.Server{
		Addr:           addr,
		Handler:        s.Router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Канал для получения ошибок
	errChan := make(chan error, 1)

	// Запускаем сервер в горутине
	go func() {
		logger.Info("Starting server", logger.Field("address", addr))
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server error", logger.Field("error", err))
			errChan <- err
		}
	}()

	// Канал для сигналов операционной системы
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Ожидаем сигнал или ошибку
	select {
	case err := <-errChan:
		logger.Error("Server failed", logger.Field("error", err))
		return err
	case sig := <-quit:
		logger.Info("Received signal", logger.Field("signal", sig))
		return s.Stop()
	}
}

func (s *Server) Stop() error {
	if s.server != nil {
		// Создаем контекст с таймаутом для graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		logger.Info("Shutting down server...")

		// Останавливаем прием новых запросов
		if err := s.server.Shutdown(ctx); err != nil {
			logger.Error("Server forced to shutdown", logger.Field("error", err))
			return err
		}
		logger.Info("Server stopped gracefully")
	}
	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Router.ServeHTTP(w, r)
}
