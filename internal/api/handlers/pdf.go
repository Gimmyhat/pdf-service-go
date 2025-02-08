package handlers

import (
	"context"
	"errors"
	"net/http"
	"pdf-service-go/internal/domain/pdf"
	"pdf-service-go/internal/pkg/circuitbreaker"
	"pdf-service-go/internal/pkg/logger"

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
}

func NewPDFHandler(service pdf.Service) *PDFHandler {
	return &PDFHandler{
		service: service,
	}
}

// ... existing code ...

func (h *PDFHandler) GenerateDocx(c *gin.Context) ([]byte, error) {
	var req pdf.DocxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Failed to parse request", zap.Error(err))
		return nil, err
	}

	return h.service.GenerateDocx(c.Request.Context(), &req)
}

func (h *PDFHandler) validateRequest(req *pdf.DocxRequest) error {
	if req.ID == "" {
		return errors.New("id is required")
	}
	if req.ApplicantType == "" {
		return errors.New("applicant type is required")
	}
	if req.ApplicantType == "ORGANIZATION" && req.OrganizationInfo == nil {
		return errors.New("organization info is required for organization applicant")
	}
	if req.ApplicantType == "INDIVIDUAL" && req.IndividualInfo == nil {
		return errors.New("individual info is required for individual applicant")
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
