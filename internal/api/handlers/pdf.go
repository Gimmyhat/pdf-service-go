package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"pdf-service-go/internal/domain/pdf"
	"pdf-service-go/internal/pkg/circuitbreaker"
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
	pdfContent, err := h.service.GenerateDocx(c.Request.Context(), &req)
	docxDuration := time.Since(docxStartTime)

	if err != nil {
		if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
			gotenbergErr = err
		} else {
			docxErr = err
		}
		logger.Error("Failed to generate PDF", zap.Error(err))
		c.JSON(h.determineErrorStatus(err), gin.H{"error": err.Error()})
		return
	}

	// Успешная генерация
	h.TrackDocxGeneration(docxDuration, false)

	// Отслеживаем размер PDF
	h.TrackPDFFile(int64(len(pdfContent)))

	totalDuration := time.Since(startTime)

	// Добавляем заголовки с метриками времени
	c.Header("X-Docx-Generation-Time", strconv.FormatFloat(docxDuration.Seconds(), 'f', 3, 64))
	c.Header("X-PDF-Conversion-Time", strconv.FormatFloat(totalDuration.Seconds()-docxDuration.Seconds(), 'f', 3, 64))
	c.Header("X-Total-Processing-Time", strconv.FormatFloat(totalDuration.Seconds(), 'f', 3, 64))

	c.Data(http.StatusOK, "application/pdf", pdfContent)
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
