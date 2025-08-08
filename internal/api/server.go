package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pdf-service-go/internal/api/middleware"
	"pdf-service-go/internal/domain/pdf"
	"pdf-service-go/internal/pkg/errortracker"
	"pdf-service-go/internal/pkg/logger"
	"pdf-service-go/internal/pkg/statistics"

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

		// Отслеживаем panic как критическую ошибку
		errortracker.TrackError(c.Request.Context(), fmt.Errorf("panic: %v", err),
			errortracker.WithComponent("api"),
			errortracker.WithErrorType("panic"),
			errortracker.WithSeverity("critical"),
			errortracker.WithHTTPStatus(http.StatusInternalServerError),
		)

		c.AbortWithStatus(http.StatusInternalServerError)
	}))

	// Добавляем middleware для захвата запросов (до логирования)
	captureConfig := statistics.RequestCaptureConfig{
		EnableCapture:     true,
		CaptureOnlyErrors: false,       // Захватываем все запросы пока тестируем
		MaxBodySize:       1024 * 1024, // 1MB максимум
		ExcludePaths:      []string{"/health", "/metrics", "/favicon.ico"},
		ExcludeHeaders:    []string{"authorization", "cookie", "x-api-key"},
		RetentionDays:     7,
		MaskSensitiveData: true,
		KeepLast:          100,
	}

	db := statistics.GetPostgresDB()
	if db != nil {
		router.Use(middleware.RequestCaptureMiddleware(db, captureConfig))
	}

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

	// API для детальной информации об ошибках
	s.Router.GET("/api/v1/errors", s.Handlers.Errors.GetErrors)
	s.Router.GET("/api/v1/errors/stats", s.Handlers.Errors.GetErrorStats)
	s.Router.GET("/api/v1/errors/:id", s.Handlers.Errors.GetErrorDetails)

	// API для анализа детальных запросов
	s.Router.GET("/api/v1/requests/error", s.Handlers.RequestAnalysis.GetErrorRequests)
	s.Router.GET("/api/v1/requests/analytics", s.Handlers.RequestAnalysis.GetErrorAnalytics)
	s.Router.GET("/api/v1/requests/:request_id", s.Handlers.RequestAnalysis.GetRequestDetail)
	s.Router.GET("/api/v1/requests/:request_id/body", s.Handlers.RequestAnalysis.GetRequestBody)
	s.Router.GET("/api/v1/requests/recent", s.Handlers.RequestAnalysis.GetRecentRequests)
	s.Router.POST("/api/v1/requests/cleanup", s.Handlers.RequestAnalysis.CleanupRequests)

	// Единый дашборд
	s.Router.GET("/dashboard", func(c *gin.Context) {
		c.File("internal/static/dashboard.html")
	})

	// Главная страница перенаправляет на дашборд
	s.Router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/dashboard")
	})

	// Веб-интерфейс для статистики (сохраняем для обратной совместимости)
	s.Router.GET("/stats", func(c *gin.Context) {
		c.File("internal/static/index.html")
	})

	// Веб-интерфейс для ошибок (сохраняем для обратной совместимости)
	s.Router.GET("/errors", func(c *gin.Context) {
		c.File("internal/static/errors.html")
	})

	// Статические файлы
	s.Router.Static("/static", "internal/static")
	// Раздача артефактов (запросов и результатов)
	s.Router.Static("/files", "/app/data/artifacts")

	// Тестовые эндпоинты для проверки логирования ошибок
	s.Router.GET("/test-error", func(c *gin.Context) {
		err := fmt.Errorf("test error for debugging")

		// Тестируем новую систему отслеживания ошибок
		errortracker.TrackError(c.Request.Context(), err,
			errortracker.WithComponent("api"),
			errortracker.WithErrorType("validation"),
			errortracker.WithSeverity("medium"),
			errortracker.WithHTTPStatus(http.StatusInternalServerError),
			errortracker.WithRequestDetails("test_mode", true),
			errortracker.WithRequestDetails("endpoint", "test-error"),
		)

		logger.Error("Test error endpoint called",
			zap.String("test_field", "test_value"),
			zap.Int("error_code", 500),
		)
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Test error message",
		})
	})

	s.Router.GET("/test-timeout", func(c *gin.Context) {
		err := fmt.Errorf("context deadline exceeded (Client.Timeout exceeded while awaiting headers)")

		errortracker.TrackError(c.Request.Context(), err,
			errortracker.WithComponent("gotenberg"),
			errortracker.WithErrorType("timeout"),
			errortracker.WithSeverity("high"),
			errortracker.WithHTTPStatus(http.StatusGatewayTimeout),
			errortracker.WithDuration(30*time.Second),
			errortracker.WithRequestDetails("pages", 5),
		)

		c.JSON(http.StatusGatewayTimeout, gin.H{
			"error": "Test timeout error",
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
		logger.Field("dashboard_ui", "/dashboard"),
		logger.Field("home_redirect", "/"),
		logger.Field("statistics_ui", "/stats"),
		logger.Field("errors_api", "/api/v1/errors"),
		logger.Field("errors_ui", "/errors"),
		logger.Field("test_endpoints", []string{"/test-error", "/test-timeout"}),
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
