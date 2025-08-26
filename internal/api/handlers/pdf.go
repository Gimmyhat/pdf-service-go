package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"pdf-service-go/internal/domain/pdf"
	"pdf-service-go/internal/pkg/circuitbreaker"
	"pdf-service-go/internal/pkg/errortracker"
	"pdf-service-go/internal/pkg/logger"
	"pdf-service-go/internal/pkg/statistics"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Определяем пользовательские ошибки
var (
	ErrInvalidRequest   = errors.New("invalid request")
	ErrProcessingFailed = errors.New("failed to process document")
)

type PDFHandler struct {
	service pdf.Service
	stats   *statistics.Statistics
}

func NewPDFHandler(service pdf.Service) *PDFHandler {
	handler := &PDFHandler{
		service: service,
	}
	handler.AddStatisticsTracking()
	return handler
}

// ... existing code ...

func (h *PDFHandler) GenerateDocx(c *gin.Context) {
	startTime := time.Now()
	var docxErr error
	var gotenbergErr error

	defer func() {
		// Отслеживаем только генерацию DOCX и конвертацию через Gotenberg
		if docxErr != nil {
			h.TrackDocxGeneration(time.Since(startTime), true)
		}
		if gotenbergErr != nil {
			h.TrackGotenbergRequest(time.Since(startTime), true)
		}
	}()

	var req pdf.DocxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Failed to parse request",
			zap.Error(err),
			zap.String("content_type", c.GetHeader("Content-Type")),
		)
		c.Header("Content-Type", "application/json; charset=utf-8")
		if err.Error() == "EOF" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "empty request body"})
			return
		}
		if strings.Contains(err.Error(), "invalid character") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request format: %v", err)})
		return
	}

	if err := h.validateRequest(&req); err != nil {
		logger.Error("Request validation failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("validation error: %v", err)})
		return
	}

	// Время начала генерации DOCX
	docxStartTime := time.Now()
	// Восстановим контекст и обогатим его путём к сохраненному payload
	ctx := c.Request.Context()
	if v, exists := c.Get("request_body_file_path"); exists {
		if s, ok := v.(string); ok && s != "" {
			ctx = context.WithValue(ctx, "request_body_file_path", s)
		}
	}
	if v, exists := c.Get("request_id"); exists {
		if s, ok := v.(string); ok && s != "" {
			ctx = context.WithValue(ctx, "request_id", s)
		}
	}
	pdfContent, err := h.service.GenerateDocx(ctx, &req)
	docxDuration := time.Since(docxStartTime)

	if err != nil {
		status := h.determineErrorStatus(err)

		// Отслеживаем ошибку с контекстом
		payloadPath := ""
		if vv, ok := ctx.Value("request_body_file_path").(string); ok {
			payloadPath = vv
		}
		stage := "docx"
		if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
			stage = "gotenberg"
		}
		errortracker.TrackError(ctx, err,
			errortracker.WithComponent("pdf"),
			errortracker.WithHTTPStatus(status),
			errortracker.WithDuration(docxDuration),
			errortracker.WithRequestDetails("pages", req.Pages),
			errortracker.WithRequestDetails("stage", stage),
			errortracker.WithRequestDetails("request_payload_path", payloadPath),
		)

		if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
			gotenbergErr = err
		} else {
			docxErr = err
		}
		logger.Error("Failed to generate PDF", zap.Error(err))
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Успешная генерация
	h.TrackDocxGeneration(docxDuration, false)

	// Сохраняем PDF в файл и обновляем статистику
	// Если сервис положил путь к timings в контекст, протянем его в gin.Context для БД
	if tp, ok := ctx.Value("timings_file_path").(string); ok && tp != "" {
		c.Set("timings_file_path", tp)
	}
	resultPath, resultSize := savePDFResultToFile(c, pdfContent)
	h.TrackPDFFile(resultSize)

	totalDuration := time.Since(startTime)

	// Добавляем заголовки с метриками времени
	c.Header("X-Docx-Generation-Time", strconv.FormatFloat(docxDuration.Seconds(), 'f', 3, 64))
	c.Header("X-PDF-Conversion-Time", strconv.FormatFloat(totalDuration.Seconds()-docxDuration.Seconds(), 'f', 3, 64))
	c.Header("X-Total-Processing-Time", strconv.FormatFloat(totalDuration.Seconds(), 'f', 3, 64))
	if resultPath != "" {
		c.Header("X-Result-File-Path", resultPath)
	}

	c.Data(http.StatusOK, "application/pdf", pdfContent)
}

// savePDFResultToFile сохраняет PDF на диск и, если есть request_id в контексте, обновляет запись о запросе
func savePDFResultToFile(c *gin.Context, pdfContent []byte) (string, int64) {
	baseDir := getArtifactsBaseDir()
	outDir := filepath.Join(baseDir, "results")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", int64(len(pdfContent))
	}
	requestIDAny, _ := c.Get("request_id")
	requestID, _ := requestIDAny.(string)
	if requestID == "" {
		requestID = fmt.Sprintf("anon_%d", time.Now().UnixNano())
	}
	filename := filepath.Join(outDir, fmt.Sprintf("%s.pdf", requestID))
	if err := os.WriteFile(filename, pdfContent, 0o644); err != nil {
		return "", int64(len(pdfContent))
	}

	// Попробуем обновить запись request_details путями к файлам
	if db := statistics.GetPostgresDB(); db != nil {
		size := int64(len(pdfContent))
		// Попытаемся также записать timings, если Python сообщил путь
		var timingsPath *string
		if tpAny, ok := c.Get("timings_file_path"); ok {
			if tp, ok2 := tpAny.(string); ok2 && tp != "" {
				timingsPath = &tp
			}
		}
		if err := db.UpdateResultFileInfoWithTimings(requestID, filename, size, timingsPath); err != nil {
			logger.Error("Failed to update result file info", zap.String("request_id", requestID), zap.Error(err))
		}
	}

	// Сохраним путь в контекст на будущее
	c.Set("result_file_path", filename)

	return filename, int64(len(pdfContent))
}

// getArtifactsBaseDir реиспользуем логику из middleware
func getArtifactsBaseDir() string {
	if v := os.Getenv("ARTIFACTS_DIR"); v != "" {
		return v
	}
	return "/app/data/artifacts"
}

func (h *PDFHandler) validateRequest(req *pdf.DocxRequest) error {
	var errors []string

	if req.ID == "" {
		errors = append(errors, "id is required")
	}
	if req.ApplicantType == "" {
		errors = append(errors, "applicant type is required")
	}
	if req.ApplicantType == "ORGANIZATION" && req.OrganizationInfo == nil {
		errors = append(errors, "organization info is required for organization applicant")
	}
	if req.ApplicantType == "INDIVIDUAL" && req.IndividualInfo == nil {
		errors = append(errors, "individual info is required for individual applicant")
	}
	if len(req.RegistryItems) == 0 {
		errors = append(errors, "at least one registry item is required")
	}
	if req.PurposeOfGeoInfoAccess == "" {
		errors = append(errors, "purpose of geo info access is required")
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
	}
	return nil
}

func (h *PDFHandler) determineErrorStatus(err error) int {
	// Здесь можно добавить более детальную обработку различных типов ошибок
	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusGatewayTimeout
	}
	if errors.Is(err, pdf.ErrTemplateNotFound) {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

// GetCircuitBreakerState возвращает текущее состояние Circuit Breaker для Gotenberg
func (h *PDFHandler) GetCircuitBreakerState() circuitbreaker.State {
	return h.service.GetCircuitBreakerState()
}

// IsCircuitBreakerHealthy возвращает true, если Circuit Breaker для Gotenberg в здоровом состоянии
func (h *PDFHandler) IsCircuitBreakerHealthy() bool {
	return h.service.IsCircuitBreakerHealthy()
}

// GetDocxGeneratorState возвращает текущее состояние Circuit Breaker для генератора DOCX
func (h *PDFHandler) GetDocxGeneratorState() circuitbreaker.State {
	return h.service.GetDocxGeneratorState()
}

// IsDocxGeneratorHealthy возвращает true, если Circuit Breaker для генератора DOCX в здоровом состоянии
func (h *PDFHandler) IsDocxGeneratorHealthy() bool {
	return h.service.IsDocxGeneratorHealthy()
}
