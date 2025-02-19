package main

import (
	"net/http"
	_ "net/http/pprof" // Импортируем pprof
	"os"
	"pdf-service-go/internal/api"
	"pdf-service-go/internal/domain/pdf"
	"pdf-service-go/internal/pkg/logger"
	"pdf-service-go/internal/pkg/statistics"
	"runtime"
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

	// Инициализируем статистику
	statsConfig := statistics.Config{
		Host:     os.Getenv("POSTGRES_HOST"),
		Port:     os.Getenv("POSTGRES_PORT"),
		DBName:   os.Getenv("POSTGRES_DB"),
		User:     os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
	}

	if err := statistics.Initialize(statsConfig); err != nil {
		logger.Fatal("Failed to initialize statistics", logger.Field("error", err))
	}
	logger.Info("Statistics initialized with PostgreSQL")

	// Создаем PDF сервис
	gotenbergURL := os.Getenv("GOTENBERG_URL")
	if gotenbergURL == "" {
		logger.Fatal("GOTENBERG_URL environment variable is not set")
	}

	pdfService := pdf.NewService(gotenbergURL)
	logger.Info("PDF service created", logger.Field("gotenberg_url", gotenbergURL))

	// Создаем обработчики
	handlers := api.NewHandlers(pdfService)
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
