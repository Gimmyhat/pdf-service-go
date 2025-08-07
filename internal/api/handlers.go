package api

import (
	"encoding/json"
	"net/http"
	"pdf-service-go/internal/api/handlers"
	"pdf-service-go/internal/domain/pdf"
	"pdf-service-go/internal/pkg/logger"
	"pdf-service-go/internal/pkg/statistics"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handlers содержит все обработчики API
type Handlers struct {
	PDF             *handlers.PDFHandler
	Statistics      *handlers.StatisticsHandler
	Errors          *handlers.ErrorHandler
	RequestAnalysis *handlers.RequestAnalysisHandler
}

// NewHandlers создает новые обработчики
func NewHandlers(service pdf.Service) *Handlers {
	return &Handlers{
		PDF:             handlers.NewPDFHandler(service),
		Statistics:      handlers.NewStatisticsHandler(),
		Errors:          handlers.NewErrorHandler(),
		RequestAnalysis: handlers.NewRequestAnalysisHandler(statistics.GetPostgresDB()),
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

	h.PDF.GenerateDocx(c)
}
