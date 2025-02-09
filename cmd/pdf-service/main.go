package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pdf-service-go/internal/pkg/logger"
	"pdf-service-go/internal/pkg/tracing"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func main() {
	// Инициализируем трейсинг
	tracingConfig := tracing.Config{
		ServiceName:    os.Getenv("OTEL_SERVICE_NAME"),
		ServiceVersion: os.Getenv("VERSION"),
		Environment:    os.Getenv("OTEL_ENVIRONMENT"),
		CollectorURL:   os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}

	shutdown, err := tracing.InitTracer(tracingConfig)
	if err != nil {
		logger.Fatal("Failed to initialize tracer", zap.Error(err))
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			logger.Error("Failed to shutdown tracer", zap.Error(err))
		}
	}()

	// Создаем роутер и оборачиваем его в middleware трейсинга
	router := http.NewServeMux()
	router.Handle("/", tracing.TracingMiddleware(http.HandlerFunc(handleRequest)))

	// Запускаем сервер
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Канал для сигналов завершения
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("Server starting", zap.String("address", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Ожидаем сигнал завершения
	<-done
	logger.Info("Server stopping...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server shutdown failed", zap.Error(err))
	}

	logger.Info("Server stopped")
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	defer span.End()

	logger.Info("Processing request",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("trace_id", span.SpanContext().TraceID().String()))

	span.AddEvent("Processing request")

	fmt.Fprintf(w, "Hello, World!")

	span.AddEvent("Request processed")
	logger.Info("Request processed successfully",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path))
}
