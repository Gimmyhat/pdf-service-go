package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pdf-service-go/internal/api/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	Router   *gin.Engine
	Handlers *Handlers
	server   *http.Server
}

func NewServer(handlers *Handlers) *Server {
	router := gin.Default()

	// Настройка лимитов
	router.MaxMultipartMemory = 8 << 20 // 8 MiB

	// Добавляем middleware для восстановления после паники
	router.Use(gin.Recovery())

	// Добавляем middleware для метрик
	router.Use(middleware.PrometheusMiddleware())

	// Добавляем middleware для таймаутов
	router.Use(func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

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
		log.Printf("Starting server on %s", addr)
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	// Канал для сигналов операционной системы
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Ожидаем сигнал или ошибку
	select {
	case err := <-errChan:
		return err
	case sig := <-quit:
		log.Printf("Received signal: %v", sig)
		return s.Stop()
	}
}

func (s *Server) Stop() error {
	if s.server != nil {
		// Создаем контекст с таймаутом для graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		log.Println("Shutting down server...")

		// Останавливаем прием новых запросов
		if err := s.server.Shutdown(ctx); err != nil {
			log.Printf("Server forced to shutdown: %v", err)
			return err
		}
	}
	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Router.ServeHTTP(w, r)
}
