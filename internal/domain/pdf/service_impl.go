package pdf

import (
	"bytes"
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
	"pdf-service-go/internal/pkg/statistics"

	"go.uber.org/zap"
)

type ServiceImpl struct {
	gotenbergClient *gotenberg.ClientWithCircuitBreaker
	docxGenerator   *docxgen.Generator
}

type StatsHandler struct {
	stats *statistics.Statistics
}

func (h *StatsHandler) TrackGotenbergRequest(duration time.Duration, hasError bool, isHealthCheck bool) {
	if !isHealthCheck {
		h.stats.TrackGotenberg(duration, hasError)
	}
}

func NewService(gotenbergURL string) Service {
	client := gotenberg.NewClientWithCircuitBreaker(gotenbergURL)
	handler := &StatsHandler{
		stats: statistics.GetInstance(),
	}
	client.SetHandler(handler)

	return &ServiceImpl{
		gotenbergClient: client,
		docxGenerator:   docxgen.NewGenerator("scripts/generate_docx.py"),
	}
}

type contextKey string

const (
	startTimeKey      contextKey = "start_time"
	docxGenerationKey contextKey = "docx_generation_time"
	pdfConversionKey  contextKey = "pdf_conversion_time"
)

func (s *ServiceImpl) GenerateDocx(ctx context.Context, req *DocxRequest) ([]byte, error) {
	log := logger.Log.With(
		zap.String("request_id", req.ID),
		zap.String("operation", req.Operation),
	)

	start := time.Now()
	var docxGenerationTime time.Duration
	var pdfConversionTime time.Duration

	// Сохраняем метрики в контекст
	ctx = context.WithValue(ctx, startTimeKey, start)
	ctx = context.WithValue(ctx, docxGenerationKey, &docxGenerationTime)
	ctx = context.WithValue(ctx, pdfConversionKey, &pdfConversionTime)

	defer func() {
		duration := time.Since(start)
		metrics.HTTPRequestDuration.WithLabelValues("POST", "/generate-pdf").Observe(duration.Seconds())
		log.Info("PDF generation completed",
			zap.Float64("duration_seconds", duration.Seconds()),
			zap.Float64("docx_generation_seconds", docxGenerationTime.Seconds()),
			zap.Float64("pdf_conversion_seconds", pdfConversionTime.Seconds()),
		)
	}()

	// Увеличиваем счетчик общего количества запросов
	metrics.RequestsTotal.WithLabelValues("started").Inc()
	log.Info("Starting PDF generation")

	// Проверяем наличие шаблона
	templatePath := filepath.Join("internal/domain/pdf/templates", "template.docx")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		log.Error("Template file not found", zap.String("path", templatePath))
		metrics.RequestsTotal.WithLabelValues("error").Inc()
		return nil, ErrTemplateNotFound
	}

	// Генерация документа в два этапа для корректного подсчета страниц
	log.Info("Starting two-phase document generation for accurate page count")

	// Этап 1: Создание черновика документа с подсчетом страниц
	draftDataFile, err := s.docxGenerator.GetTempManager().CreateTemp(ctx, fmt.Sprintf("draft-data-%d-%x-*.json", time.Now().UnixNano(), time.Now().Nanosecond()))
	if err != nil {
		log.Error("Failed to create temp draft JSON file", zap.Error(err))
		metrics.RequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to create temp draft JSON file: %w", err)
	}
	defer draftDataFile.Close()
	defer os.Remove(draftDataFile.Name())

	draftDocxFile, err := s.docxGenerator.GetTempManager().CreateTemp(ctx, fmt.Sprintf("draft-docx-%d-%x-*.docx", time.Now().UnixNano(), time.Now().Nanosecond()))
	if err != nil {
		log.Error("Failed to create temp draft DOCX file", zap.Error(err))
		metrics.RequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to create temp draft DOCX file: %w", err)
	}
	defer draftDocxFile.Close()
	defer os.Remove(draftDocxFile.Name())

	// Устанавливаем временное значение для страниц и сохраняем во временный JSON
	reqCopy := *req
	reqCopy.Pages = 0      // Указываем, что это черновик для подсчета
	reqCopy.IsDraft = true // Флаг, указывающий что это черновик

	draftData, err := json.Marshal(reqCopy)
	if err != nil {
		log.Error("Failed to marshal draft request data", zap.Error(err))
		metrics.RequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to marshal draft request data: %w", err)
	}

	if _, err = draftDataFile.Write(draftData); err != nil {
		log.Error("Failed to write draft data file", zap.Error(err))
		metrics.RequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to write draft data file: %w", err)
	}

	// Генерируем черновик DOCX
	log.Info("Generating draft DOCX for page counting")
	if err := s.docxGenerator.Generate(ctx, templatePath, draftDataFile.Name(), draftDocxFile.Name()); err != nil {
		log.Error("Failed to generate draft DOCX", zap.Error(err))
		metrics.RequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to generate draft DOCX: %w", err)
	}

	// Конвертируем черновик в PDF и подсчитываем страницы
	log.Info("Converting draft DOCX to PDF for page counting")
	draftPdfContent, err := s.gotenbergClient.ConvertDocxToPDF(draftDocxFile.Name())
	if err != nil {
		log.Error("Failed to convert draft DOCX to PDF", zap.Error(err))
		metrics.RequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to convert draft DOCX to PDF: %w", err)
	}

	// Подсчитываем количество страниц в PDF
	pageCount := countPages(draftPdfContent)
	log.Info("Counted pages in draft PDF", zap.Int("pageCount", pageCount))

	// Этап 2: Создание финального документа с правильным количеством страниц
	dataFile, err := s.docxGenerator.GetTempManager().CreateTemp(ctx, fmt.Sprintf("data-%d-%x-*.json", time.Now().UnixNano(), time.Now().Nanosecond()))
	if err != nil {
		log.Error("Failed to create temp JSON file", zap.Error(err))
		metrics.RequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to create temp JSON file: %w", err)
	}
	defer dataFile.Close()
	defer os.Remove(dataFile.Name())

	docxFile, err := s.docxGenerator.GetTempManager().CreateTemp(ctx, fmt.Sprintf("docx-%d-%x-*.docx", time.Now().UnixNano(), time.Now().Nanosecond()))
	if err != nil {
		log.Error("Failed to create temp DOCX file", zap.Error(err))
		metrics.RequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to create temp DOCX file: %w", err)
	}
	defer docxFile.Close()
	defer os.Remove(docxFile.Name())

	// Устанавливаем правильное значение для страниц в оригинальном запросе
	req.Pages = pageCount
	req.IsDraft = false

	// Сохраняем данные во временный JSON файл
	data, err := json.Marshal(req)
	if err != nil {
		log.Error("Failed to marshal request data", zap.Error(err))
		metrics.RequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	if _, err = dataFile.Write(data); err != nil {
		log.Error("Failed to write data file", zap.Error(err))
		metrics.RequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to write data file: %w", err)
	}

	// Генерируем финальный DOCX
	log.Info("Starting final DOCX generation", zap.Int("pages", req.Pages))
	docxStart := time.Now()
	if err := s.docxGenerator.Generate(ctx, templatePath, dataFile.Name(), docxFile.Name()); err != nil {
		log.Error("Failed to generate DOCX", zap.Error(err))
		metrics.RequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to generate DOCX: %w", err)
	}
	docxGenerationTime = time.Since(docxStart)

	// Конвертируем DOCX в PDF через Gotenberg
	log.Info("Starting PDF conversion with Gotenberg")
	pdfStart := time.Now()
	pdfContent, err := s.gotenbergClient.ConvertDocxToPDF(docxFile.Name())
	pdfConversionTime = time.Since(pdfStart)

	if err != nil {
		log.Error("Failed to convert to PDF", zap.Error(err))
		metrics.RequestsTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to convert to PDF: %w", err)
	}

	// После получения ответа от Gotenberg
	log.Info("PDF conversion completed",
		zap.Float64("docx_generation_seconds", docxGenerationTime.Seconds()),
		zap.Float64("pdf_conversion_seconds", pdfConversionTime.Seconds()),
		zap.Float64("pdf_size_mb", float64(len(pdfContent))/1024/1024),
	)

	// Успешное завершение
	metrics.RequestsTotal.WithLabelValues("completed").Inc()
	metrics.PDFFileSizeBytes.WithLabelValues("generate-pdf").Observe(float64(len(pdfContent)))

	return pdfContent, nil
}

// Функция для подсчета страниц в PDF
func countPages(pdfContent []byte) int {
	// Простой подсчет количества вхождений "/Page" в PDF
	pageCount := bytes.Count(pdfContent, []byte("/Page"))

	// Логика корректировки подсчета:
	// 1. Если страниц меньше 4, указываем минимум 1 лист
	// 2. Иначе вычитаем 3, предполагая что это служебные страницы
	if pageCount < 4 {
		return 1
	}

	// Вычитаем 3 страницы из общего количества
	return pageCount - 3
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
