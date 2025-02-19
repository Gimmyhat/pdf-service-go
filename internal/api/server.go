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
	"pdf-service-go/internal/domain/pdf"
	"pdf-service-go/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Server struct {
	Router   *gin.Engine
	Handlers *Handlers
	server   *http.Server
	mux      *http.ServeMux
	service  pdf.Service
}

func NewServer(handlers *Handlers, service pdf.Service) *Server {
	// Отключаем стандартный логгер gin
	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()

	// Создаем новый роутер без стандартного логгера
	router := gin.New()

	// Настройка лимитов
	router.MaxMultipartMemory = 8 << 20 // 8 MiB
	logger.Info("Server memory limits configured", zap.String("max_multipart_memory", "8 MiB"))

	// Добавляем middleware для восстановления после паники с использованием zap
	router.Use(gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, err interface{}) {
		logger.Error("Panic recovered",
			zap.Any("error", err),
			zap.String("path", c.Request.URL.Path),
		)
		c.AbortWithStatus(http.StatusInternalServerError)
	}))

	// Добавляем middleware для логирования с использованием zap
	router.Use(func(c *gin.Context) {
		// Пропускаем логирование для health check
		if c.Request.URL.Path != "/health" {
			start := time.Now()
			path := c.Request.URL.Path
			query := c.Request.URL.RawQuery

			c.Next()

			latency := time.Since(start)
			statusCode := c.Writer.Status()

			if statusCode >= 400 {
				// Логируем ошибки с уровнем Error
				logger.Error("HTTP Request",
					zap.String("client_ip", c.ClientIP()),
					zap.String("method", c.Request.Method),
					zap.String("path", path),
					zap.String("query", query),
					zap.Int("status_code", statusCode),
					zap.Duration("latency", latency),
					zap.String("error", c.Errors.String()),
				)
			} else {
				// Логируем успешные запросы с уровнем Info
				logger.Info("HTTP Request",
					zap.String("client_ip", c.ClientIP()),
					zap.String("method", c.Request.Method),
					zap.String("path", path),
					zap.String("query", query),
					zap.Int("status_code", statusCode),
					zap.Duration("latency", latency),
				)
			}
		} else {
			c.Next()
		}
	})

	// Добавляем middleware для метрик и статистики
	router.Use(middleware.PrometheusMiddleware())
	router.Use(middleware.StatisticsMiddleware())

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
		mux:      http.NewServeMux(),
		service:  service,
	}
}

func (s *Server) SetupRoutes() {
	// Health check для k8s
	s.Router.GET("/health", s.handleHealth())

	// Метрики Prometheus
	s.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Статистика API
	s.Router.GET("/api/v1/statistics", s.Handlers.Statistics.GetStatistics)

	// Веб-интерфейс для статистики
	s.Router.GET("/stats", func(c *gin.Context) {
		c.File("internal/static/index.html")
	})

	// Статические файлы
	s.Router.Static("/static", "internal/static")

	// Тестовый эндпоинт для проверки логирования ошибок
	s.Router.GET("/test-error", func(c *gin.Context) {
		logger.Error("Test error endpoint called",
			zap.String("test_field", "test_value"),
			zap.Int("error_code", 500),
		)
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Test error message",
		})
	})

	// API endpoints
	v1 := s.Router.Group("/api/v1")
	{
		v1.POST("/docx", func(c *gin.Context) {
			s.Handlers.PDF.GenerateDocx(c)
		})
	}

	// Поддержка старого endpoint'а для обратной совместимости
	s.Router.POST("/generate-pdf", func(c *gin.Context) {
		s.Handlers.PDF.GenerateDocx(c)
	})

	logger.Info("Routes configured",
		logger.Field("health_endpoint", "/health"),
		logger.Field("metrics_endpoint", "/metrics"),
		logger.Field("statistics_endpoint", "/api/v1/statistics"),
		logger.Field("statistics_ui", "/stats"),
		logger.Field("test_endpoint", "/test-error"),
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

func (s *Server) handleHealth() gin.HandlerFunc {
	return func(c *gin.Context) {
		gotenbergState := s.service.GetCircuitBreakerState()
		docxState := s.service.GetDocxGeneratorState()
		isHealthy := s.service.IsCircuitBreakerHealthy() && s.service.IsDocxGeneratorHealthy()

		status := "healthy"
		if !isHealthy {
			status = "unhealthy"
		}

		response := gin.H{
			"status":    status,
			"timestamp": time.Now().Format(time.RFC3339),
			"details": gin.H{
				"circuit_breakers": gin.H{
					"gotenberg": gin.H{
						"status": s.service.IsCircuitBreakerHealthy(),
						"state":  gotenbergState.String(),
					},
					"docx_generator": gin.H{
						"status": s.service.IsDocxGeneratorHealthy(),
						"state":  docxState.String(),
					},
				},
			},
		}

		c.Header("Content-Type", "application/json; charset=utf-8")
		if !isHealthy {
			c.JSON(http.StatusServiceUnavailable, response)
			return
		}

		c.JSON(http.StatusOK, response)
	}
}
