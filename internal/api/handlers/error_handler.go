package handlers

import (
	"net/http"
	"strconv"
	"time"

	"pdf-service-go/internal/pkg/statistics"

	"github.com/gin-gonic/gin"
)

// ErrorHandler обработчик для детальной информации об ошибках
type ErrorHandler struct {
	stats *statistics.Statistics
}

// NewErrorHandler создает новый обработчик ошибок
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		stats: statistics.GetInstance(),
	}
}

// GetErrors возвращает детальную информацию об ошибках
func (h *ErrorHandler) GetErrors(c *gin.Context) {
	// Параметры запроса
	limitStr := c.DefaultQuery("limit", "50")
	periodStr := c.DefaultQuery("period", "24h")
	errorType := c.Query("type")
	component := c.Query("component")
	severity := c.Query("severity")

	// Парсим лимит
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 1000 {
		limit = 50
	}

	// Парсим период
	var since time.Time
	switch periodStr {
	case "1h":
		since = time.Now().Add(-1 * time.Hour)
	case "6h":
		since = time.Now().Add(-6 * time.Hour)
	case "24h":
		since = time.Now().Add(-24 * time.Hour)
	case "7d":
		since = time.Now().Add(-7 * 24 * time.Hour)
	case "30d":
		since = time.Now().Add(-30 * 24 * time.Hour)
	default:
		since = time.Now().Add(-24 * time.Hour)
	}

	// Получаем сводку ошибок
	errorSummary, err := h.stats.GetErrorSummary(since, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get error summary: " + err.Error(),
		})
		return
	}

	// Фильтруем результаты если указаны фильтры
	if errorType != "" || component != "" || severity != "" {
		errorSummary.RecentErrors = filterErrors(errorSummary.RecentErrors, errorType, component, severity)
		errorSummary.ErrorPatterns = filterPatterns(errorSummary.ErrorPatterns, errorType, component)
	}

	// Добавляем метаинформацию
	response := gin.H{
		"summary": errorSummary,
		"period":  periodStr,
		"since":   since.Format(time.RFC3339),
		"limit":   limit,
		"filters": gin.H{
			"type":      errorType,
			"component": component,
			"severity":  severity,
		},
		"available_filters": gin.H{
			"types":      []string{"timeout", "validation", "connection", "resource", "panic", "unknown"},
			"components": []string{"gotenberg", "docx", "pdf", "api", "database", "system"},
			"severities": []string{"critical", "high", "medium", "low"},
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetErrorDetails возвращает детали конкретной ошибки
func (h *ErrorHandler) GetErrorDetails(c *gin.Context) {
	errorID := c.Param("id")
	if errorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error ID is required"})
		return
	}

	// Здесь можно добавить метод для получения конкретной ошибки по ID
	// Пока возвращаем заглушку
	c.JSON(http.StatusOK, gin.H{
		"message": "Error details endpoint - to be implemented",
		"id":      errorID,
	})
}

// GetErrorStats возвращает статистику ошибок
func (h *ErrorHandler) GetErrorStats(c *gin.Context) {
	periodStr := c.DefaultQuery("period", "24h")

	var since time.Time
	switch periodStr {
	case "1h":
		since = time.Now().Add(-1 * time.Hour)
	case "6h":
		since = time.Now().Add(-6 * time.Hour)
	case "24h":
		since = time.Now().Add(-24 * time.Hour)
	case "7d":
		since = time.Now().Add(-7 * 24 * time.Hour)
	default:
		since = time.Now().Add(-24 * time.Hour)
	}

	patterns, err := h.stats.GetDB().GetErrorPatterns(since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get error patterns: " + err.Error(),
		})
		return
	}

	total, last24h, lastHour, err := h.stats.GetDB().GetErrorCounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get error counts: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"period":        periodStr,
		"since":         since.Format(time.RFC3339),
		"total_errors":  total,
		"errors_24h":    last24h,
		"errors_1h":     lastHour,
		"patterns":      patterns,
		"health_status": determineHealthStatus(last24h, lastHour),
	})
}

// Вспомогательные функции
func filterErrors(errors []statistics.ErrorDetails, errorType, component, severity string) []statistics.ErrorDetails {
	var filtered []statistics.ErrorDetails
	for _, e := range errors {
		if errorType != "" && e.ErrorType != errorType {
			continue
		}
		if component != "" && e.Component != component {
			continue
		}
		if severity != "" && e.Severity != severity {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}

func filterPatterns(patterns []statistics.ErrorPattern, errorType, component string) []statistics.ErrorPattern {
	var filtered []statistics.ErrorPattern
	for _, p := range patterns {
		if errorType != "" && p.ErrorType != errorType {
			continue
		}
		if component != "" && p.Component != component {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered
}

func determineHealthStatus(errors24h, errors1h int64) string {
	if errors1h > 10 {
		return "critical"
	}
	if errors1h > 5 {
		return "warning"
	}
	if errors24h > 50 {
		return "degraded"
	}
	return "healthy"
}
