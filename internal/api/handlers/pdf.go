package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"pdf-service-go/internal/domain/pdf"
	"pdf-service-go/internal/metrics"

	"github.com/gin-gonic/gin"
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

func (h *PDFHandler) GenerateDocx(c *gin.Context) {
	// Проверяем размер тела запроса
	if c.Request.ContentLength > 10<<20 { // 10 MB
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error":    "request body too large",
			"max_size": "10MB",
		})
		return
	}

	var req pdf.DocxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   ErrInvalidRequest.Error(),
			"details": err.Error(),
		})
		return
	}

	// Проверяем обязательные поля
	if err := h.validateRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   ErrInvalidRequest.Error(),
			"details": err.Error(),
		})
		return
	}

	// Используем контекст с таймаутом из middleware
	pdfBytes, err := h.service.GenerateDocx(c.Request.Context(), &req)
	if err != nil {
		// Определяем тип ошибки и возвращаем соответствующий статус
		status := h.determineErrorStatus(err)
		c.JSON(status, gin.H{
			"error":   ErrProcessingFailed.Error(),
			"details": err.Error(),
		})
		return
	}

	// Записываем метрику размера файла
	metrics.FileSize.WithLabelValues("pdf").Observe(float64(len(pdfBytes)))

	filename := fmt.Sprintf("%s.pdf", req.ID)

	// Проверяем размер сгенерированного файла
	if len(pdfBytes) > 50<<20 { // 50 MB
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":    "generated file too large",
			"max_size": "50MB",
		})
		return
	}

	// Устанавливаем правильные заголовки для скачивания PDF файла
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Length", fmt.Sprint(len(pdfBytes)))
	c.Header("Cache-Control", "private")
	c.Header("Pragma", "public")

	c.Data(http.StatusOK, "application/pdf", pdfBytes)
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
