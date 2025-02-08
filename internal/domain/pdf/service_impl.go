package pdf

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"pdf-service-go/internal/pkg/circuitbreaker"
	"pdf-service-go/internal/pkg/docxgen"
	"pdf-service-go/internal/pkg/gotenberg"
	"pdf-service-go/internal/pkg/logger"
	"pdf-service-go/internal/pkg/metrics"

	"go.uber.org/zap"
)

type ServiceImpl struct {
	gotenbergClient *gotenberg.ClientWithCircuitBreaker
	docxGenerator   *docxgen.Generator
}

func NewService(gotenbergURL string) Service {
	return &ServiceImpl{
		gotenbergClient: gotenberg.NewClientWithCircuitBreaker(gotenbergURL),
		docxGenerator:   docxgen.NewGenerator("scripts/generate_docx.py"),
	}
}

func (s *ServiceImpl) GenerateDocx(ctx context.Context, req *DocxRequest) ([]byte, error) {
	log := logger.Log.With(
		zap.String("request_id", req.ID),
		zap.String("operation", req.Operation),
	)

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.PDFGenerationDuration.WithLabelValues(req.ID).Observe(duration)
		log.Info("PDF generation completed",
			zap.Float64("duration_seconds", duration),
		)
	}()

	// Увеличиваем счетчик начала генерации
	metrics.PDFGenerationTotal.WithLabelValues("started").Inc()
	log.Info("Starting PDF generation")

	// Проверяем наличие шаблона
	templatePath := filepath.Join("internal/domain/pdf/templates", "template.docx")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		log.Error("Template file not found", zap.String("path", templatePath))
		metrics.PDFGenerationTotal.WithLabelValues("error").Inc()
		return nil, ErrTemplateNotFound
	}

	// Создаем временные файлы
	tempDir := os.TempDir()
	dataPath := filepath.Join(tempDir, fmt.Sprintf("%s.json", req.ID))
	docxPath := filepath.Join(tempDir, fmt.Sprintf("%s.docx", req.ID))
	defer os.Remove(dataPath)
	defer os.Remove(docxPath)

	log.Debug("Created temporary files",
		zap.String("data_path", dataPath),
		zap.String("docx_path", docxPath),
	)

	// Сохраняем данные во временный JSON файл
	data, err := json.Marshal(req)
	if err != nil {
		log.Error("Failed to marshal request data", zap.Error(err))
		metrics.PDFGenerationTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}
	err = os.WriteFile(dataPath, data, 0644)
	if err != nil {
		log.Error("Failed to write data file", zap.Error(err))
		metrics.PDFGenerationTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to write data file: %w", err)
	}

	// Генерируем DOCX с использованием нового генератора
	log.Info("Starting DOCX generation")
	if err := s.docxGenerator.Generate(ctx, templatePath, dataPath, docxPath); err != nil {
		log.Error("Failed to generate DOCX", zap.Error(err))
		metrics.PDFGenerationTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to generate DOCX: %w", err)
	}

	// Начало запроса к Gotenberg
	log.Info("Starting PDF conversion with Gotenberg")
	gotenbergStart := time.Now()

	// Конвертируем DOCX в PDF через Gotenberg
	pdfContent, err := s.gotenbergClient.ConvertDocxToPDF(docxPath)
	if err != nil {
		log.Error("Failed to convert to PDF", zap.Error(err))
		metrics.PDFGenerationTotal.WithLabelValues("error").Inc()
		metrics.GotenbergRequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to convert to PDF: %w", err)
	}

	// После получения ответа от Gotenberg
	gotenbergDuration := time.Since(gotenbergStart).Seconds()
	metrics.GotenbergRequestDuration.WithLabelValues("convert").Observe(gotenbergDuration)
	metrics.GotenbergRequestsTotal.WithLabelValues("success").Inc()

	log.Info("PDF conversion completed",
		zap.Float64("gotenberg_duration_seconds", gotenbergDuration),
		zap.Float64("pdf_size_mb", float64(len(pdfContent))/1024/1024),
	)

	// Успешное завершение
	metrics.PDFGenerationTotal.WithLabelValues("success").Inc()
	metrics.PDFFileSizeBytes.WithLabelValues(req.ID).Observe(float64(len(pdfContent)))

	return pdfContent, nil
}

// GetCircuitBreakerState возвращает текущее состояние Circuit Breaker для Gotenberg
func (s *ServiceImpl) GetCircuitBreakerState() circuitbreaker.State {
	return s.gotenbergClient.State()
}

// IsCircuitBreakerHealthy возвращает true, если Circuit Breaker для Gotenberg в здоровом состоянии
func (s *ServiceImpl) IsCircuitBreakerHealthy() bool {
	return s.gotenbergClient.IsHealthy()
}

// GetDocxGeneratorState возвращает текущее состояние Circuit Breaker для генератора DOCX
func (s *ServiceImpl) GetDocxGeneratorState() circuitbreaker.State {
	return s.docxGenerator.State()
}

// IsDocxGeneratorHealthy возвращает true, если Circuit Breaker для генератора DOCX в здоровом состоянии
func (s *ServiceImpl) IsDocxGeneratorHealthy() bool {
	return s.docxGenerator.IsHealthy()
}
