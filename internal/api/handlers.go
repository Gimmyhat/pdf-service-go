package api

import (
	"encoding/json"
	"net/http"
	"pdf-service-go/internal/api/handlers"
	"pdf-service-go/internal/domain/pdf"
	"pdf-service-go/internal/pkg/logger"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handlers struct {
	PDF *handlers.PDFHandler
}

func NewHandlers(pdfHandler *handlers.PDFHandler) *Handlers {
	return &Handlers{
		PDF: pdfHandler,
	}
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	gotenbergState := h.PDF.GetCircuitBreakerState()
	docxState := h.PDF.GetDocxGeneratorState()
	isHealthy := h.PDF.IsCircuitBreakerHealthy() && h.PDF.IsDocxGeneratorHealthy()

	status := "healthy"
	if !isHealthy {
		status = "unhealthy"
	}

	response := gin.H{
		"status":    status,
		"timestamp": time.Now().Format(time.RFC3339),
		"details": gin.H{
			"circuit_breakers": gin.H{
				"gotenberg": gin.H{
					"status": h.PDF.IsCircuitBreakerHealthy(),
					"state":  gotenbergState.String(),
				},
				"docx_generator": gin.H{
					"status": h.PDF.IsDocxGeneratorHealthy(),
					"state":  docxState.String(),
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if !isHealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode health response", zap.Error(err))
	}
}

func (h *Handlers) GenerateDocx(c *gin.Context) {
	var req pdf.DocxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Failed to parse request", zap.Error(err))
		c.JSON(400, gin.H{"error": "Invalid request format"})
		return
	}

	pdfContent, err := h.PDF.GenerateDocx(c)
	if err != nil {
		logger.Error("Failed to generate PDF", zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename=result.pdf")
	c.Data(200, "application/pdf", pdfContent)
}
