package main

import (
	"context"
	"fmt"
	_ "net/http/pprof" // Импортируем pprof
	"os"
	"os/signal"
	"pdf-service-go/internal/api/handlers"
	"pdf-service-go/internal/api/middleware"
	"pdf-service-go/internal/api/server"
	"pdf-service-go/internal/domain/pdf"
	"pdf-service-go/internal/pkg/cache"
	"pdf-service-go/internal/pkg/config"
	"pdf-service-go/internal/pkg/gotenberg"
	"pdf-service-go/internal/pkg/logger"
	"pdf-service-go/internal/pkg/statistics"
	"runtime"
	"syscall"
	"time"
)

func main() {
	// Включаем сборку профилей CPU и Memory
	runtime.SetMutexProfileFraction(1)
	runtime.SetBlockProfileRate(1)

	// Инициализируем логгер в самом начале
	if err := logger.InitLogger(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := logger.Log.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
		}
	}()

	// Теперь можно использовать логгер
	logger.Info("Starting PDF service...")

	// Загрузка конфигурации
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("Failed to load config", err)
	}

	// Инициализация зависимостей
	ctx := context.Background()

	// Инициализация кэша
	cache, err := cache.NewCache(cfg.Cache)
	if err != nil {
		logger.Fatal("Failed to initialize cache", err)
	}

	// Инициализация Gotenberg клиента
	gotenbergClient, err := gotenberg.NewClientWithPool(cfg.Gotenberg)
	if err != nil {
		logger.Fatal("Failed to initialize Gotenberg client", err)
	}

	// Инициализация статистики
	stats, err := statistics.NewPostgresStatistics(cfg.Statistics)
	if err != nil {
		logger.Fatal("Failed to initialize statistics", err)
	}

	// Инициализация сервиса PDF
	pdfService := pdf.NewServiceImpl(cache, gotenbergClient, stats)

	// Инициализация обработчиков
	handlers := handlers.NewHandlers(pdfService, stats)

	// Инициализация middleware
	middleware := middleware.NewMiddleware(stats)

	// Инициализация сервера
	srv := server.NewServer(cfg.Server, handlers, middleware)

	// Запуск сервера в горутине
	go func() {
		if err := srv.Start(); err != nil {
			logger.Error("Server error", err)
		}
	}()

	// Ожидание сигнала для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Создаем контекст с таймаутом для graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Останавливаем сервер
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", err)
	}

	logger.Info("Server exiting")
}
