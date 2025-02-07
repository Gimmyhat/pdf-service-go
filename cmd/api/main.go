package main

import (
	"log"
	"os"
	"pdf-service-go/internal/api"
	"pdf-service-go/internal/api/handlers"
	"pdf-service-go/internal/domain/pdf"
)

func main() {
	// Получаем URL Gotenberg из переменной окружения
	gotenbergURL := os.Getenv("GOTENBERG_API_URL")
	if gotenbergURL == "" {
		gotenbergURL = "http://gotenberg:3000" // значение по умолчанию
	}

	// Создаем сервисы
	pdfService := pdf.NewService(gotenbergURL)

	// Создаем обработчики
	pdfHandler := handlers.NewPDFHandler(pdfService)
	handlers := api.NewHandlers(pdfHandler)

	// Создаем и настраиваем сервер
	server := api.NewServer(handlers)
	server.SetupRoutes()

	// Запускаем сервер
	if err := server.Start(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
