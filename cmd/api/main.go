package main

import (
	"context"
	"os"
	"pdf-service-go/internal/api"
	"pdf-service-go/internal/api/handlers"
	"pdf-service-go/internal/domain/pdf"
	"pdf-service-go/internal/pkg/logger"
	"pdf-service-go/internal/pkg/tracing"
	"time"
)

func main() {
	// Инициализируем логгер
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	if err := logger.Init(logLevel); err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Log.Sync()

	// Инициализируем трейсинг
	tracingConfig := tracing.Config{
		ServiceName:    os.Getenv("OTEL_SERVICE_NAME"),
		ServiceVersion: os.Getenv("VERSION"),
		Environment:    os.Getenv("OTEL_ENVIRONMENT"),
		CollectorURL:   os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}

	shutdown, err := tracing.InitTracer(tracingConfig)
	if err != nil {
		logger.Fatal("Failed to initialize tracer", logger.Field("error", err))
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			logger.Error("Failed to shutdown tracer", logger.Field("error", err))
		}
	}()

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

	// Запускаем сервер
	logger.Info("Starting server", logger.Field("address", ":8080"))
	if err := server.Start(":8080"); err != nil {
		logger.Fatal("Failed to start server", logger.Field("error", err))
	}
}
