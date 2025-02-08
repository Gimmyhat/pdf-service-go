package main

import (
	"os"
	"pdf-service-go/internal/api"
	"pdf-service-go/internal/api/handlers"
	"pdf-service-go/internal/domain/pdf"
	"pdf-service-go/internal/pkg/logger"
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
	server := api.NewServer(handlers)
	server.SetupRoutes()
	logger.Info("Server configured and routes set up")

	// Запускаем сервер
	logger.Info("Starting server", logger.Field("address", ":8080"))
	if err := server.Start(":8080"); err != nil {
		logger.Fatal("Failed to start server", logger.Field("error", err))
	}
}
