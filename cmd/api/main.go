package main

import (
	"context"
	"net/http"
	_ "net/http/pprof" // Импортируем pprof
	"os"
	"pdf-service-go/internal/api"
	"pdf-service-go/internal/api/handlers"
	"pdf-service-go/internal/domain/pdf"
	"pdf-service-go/internal/pkg/logger"
	"pdf-service-go/internal/pkg/tracing"
	"runtime"
	"time"
)

func main() {
	// Включаем сборку профилей CPU и Memory
	runtime.SetMutexProfileFraction(1)
	runtime.SetBlockProfileRate(1)

	// Инициализируем логгер
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	if err := logger.Init(logLevel); err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Log.Sync()

	var shutdown func(context.Context) error

	// Инициализируем трейсинг только если указан коллектор
	if collectorURL := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); collectorURL != "" {
		tracingConfig := tracing.Config{
			ServiceName:    os.Getenv("OTEL_SERVICE_NAME"),
			ServiceVersion: os.Getenv("VERSION"),
			Environment:    os.Getenv("OTEL_ENVIRONMENT"),
			CollectorURL:   collectorURL,
		}

		var err error
		shutdown, err = tracing.InitTracer(tracingConfig)
		if err != nil {
			logger.Error("Failed to initialize tracer", logger.Field("error", err))
		} else {
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := shutdown(ctx); err != nil {
					logger.Error("Failed to shutdown tracer", logger.Field("error", err))
				}
			}()
		}
	}

	// Получаем URL Gotenberg из переменной окружения
	gotenbergURL := os.Getenv("GOTENBERG_API_URL")
	if gotenbergURL == "" {
		gotenbergURL = "http://gotenberg:3000" // значение по умолчанию
		logger.Info("Using default Gotenberg URL", logger.Field("url", gotenbergURL))
	}

	// Создаем сервисы
	pdfService := pdf.NewService(gotenbergURL)
	logger.Info("PDF service created", logger.Field("gotenberg_url", gotenbergURL))

	// Создаем обработчики
	pdfHandler := handlers.NewPDFHandler(pdfService)
	handlers := api.NewHandlers(pdfHandler)
	logger.Info("Handlers initialized")

	// Создаем и настраиваем сервер
	server := api.NewServer(handlers, pdfService)
	server.SetupRoutes()
	logger.Info("Server configured and routes set up")

	// Запускаем pprof сервер на отдельном порту
	go func() {
		pprofPort := os.Getenv("PPROF_PORT")
		if pprofPort == "" {
			pprofPort = "6060"
		}
		logger.Info("Starting pprof server", logger.Field("port", pprofPort))
		if err := http.ListenAndServe(":"+pprofPort, nil); err != nil {
			logger.Error("Failed to start pprof server", logger.Field("error", err))
		}
	}()

	// Получаем порт для основного сервера из переменной окружения
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	// Запускаем основной сервер
	if err := server.Start(":" + serverPort); err != nil {
		logger.Fatal("Failed to start server", logger.Field("error", err))
	}
}
