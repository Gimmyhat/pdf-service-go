package handlers

import (
	"net/http"
	"os"
	"pdf-service-go/internal/pkg/logger"
	"pdf-service-go/internal/pkg/statistics"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type RequestAnalysisHandler struct {
	db *statistics.PostgresDB
}

func NewRequestAnalysisHandler(db *statistics.PostgresDB) *RequestAnalysisHandler {
	return &RequestAnalysisHandler{db: db}
}

// GetRequestDetail возвращает детальную информацию о конкретном запросе
func (h *RequestAnalysisHandler) GetRequestDetail(c *gin.Context) {
	requestID := c.Param("request_id")
	if requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "request_id is required",
		})
		return
	}

	detail, err := h.db.GetRequestDetail(requestID)
	if err != nil {
		logger.Error("Failed to get request detail",
			zap.String("request_id", requestID),
			zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Request not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"request_detail": detail,
	})
}

// GetErrorRequests возвращает запросы с ошибками
func (h *RequestAnalysisHandler) GetErrorRequests(c *gin.Context) {
	// Парсим параметры
	limitStr := c.DefaultQuery("limit", "20")
	periodStr := c.DefaultQuery("period", "24h")
	categoryFilter := c.Query("category")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	// Парсим период
	since := time.Now().Add(-24 * time.Hour) // по умолчанию 24 часа
	if periodStr != "" {
		duration, err := time.ParseDuration(periodStr)
		if err == nil {
			since = time.Now().Add(-duration)
		}
	}

	var details []statistics.RequestDetail

	if categoryFilter != "" {
		// Фильтр по категории ошибки
		details, err = h.db.GetRequestDetailsByPattern(categoryFilter, limit, since)
	} else {
		// Все ошибки
		details, err = h.db.GetRequestDetailsByError(limit, since)
	}

	if err != nil {
		logger.Error("Failed to get error requests", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve error requests",
		})
		return
	}

	// Подготавливаем ответ с дополнительной аналитикой
	response := gin.H{
		"error_requests": details,
		"total_found":    len(details),
		"period":         periodStr,
		"since":          since,
		"filters": gin.H{
			"category": categoryFilter,
			"limit":    limit,
		},
	}

	// Добавляем аналитику по категориям ошибок
	if len(details) > 0 {
		categoryStats := make(map[string]int)
		pathStats := make(map[string]int)
		statusStats := make(map[int]int)

		for _, detail := range details {
			if detail.ErrorCategory != "" {
				categoryStats[detail.ErrorCategory]++
			}
			pathStats[detail.Path]++
			statusStats[detail.HTTPStatus]++
		}

		response["analytics"] = gin.H{
			"by_category":    categoryStats,
			"by_path":        pathStats,
			"by_status_code": statusStats,
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetRecentRequests возвращает последние запросы (успешные и с ошибками)
func (h *RequestAnalysisHandler) GetRecentRequests(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 1000 {
		limit = 100
	}

	details, err := h.db.GetRecentRequests(limit)
	if err != nil {
		logger.Error("Failed to get recent requests", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve recent requests"})
		return
	}

	// Преобразуем абсолютные пути к публичным URL через /files
	baseDir := getArtifactsBaseDir()
	makePublic := func(p *string) *string {
		if p == nil || *p == "" {
			return p
		}
		path := *p
		// На Windows могут быть обратные слеши — нормализуем
		path = strings.ReplaceAll(path, "\\", "/")
		b := strings.ReplaceAll(baseDir, "\\", "/")
		if strings.HasPrefix(path, b) {
			rel := strings.TrimPrefix(path, b)
			if !strings.HasPrefix(rel, "/") {
				rel = "/" + rel
			}
			url := "/files" + rel
			return &url
		}
		return p
	}

	for i := range details {
		details[i].RequestFilePath = makePublic(details[i].RequestFilePath)
		details[i].ResultFilePath = makePublic(details[i].ResultFilePath)
	}

	c.JSON(http.StatusOK, gin.H{
		"recent_requests": details,
		"total":           len(details),
	})
}

// CleanupRequests запускает очистку артефактов, оставляя только последние keep записей
func (h *RequestAnalysisHandler) CleanupRequests(c *gin.Context) {
	keepStr := c.DefaultQuery("keep", "100")
	keep, err := strconv.Atoi(keepStr)
	if err != nil || keep <= 0 {
		keep = 100
	}
	if err := h.db.CleanupOldRequestArtifactsKeepLast(keep); err != nil {
		logger.Error("Failed to cleanup request artifacts", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cleanup failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "kept": keep})
}

// GetRequestBody возвращает тело конкретного запроса
func (h *RequestAnalysisHandler) GetRequestBody(c *gin.Context) {
	requestID := c.Param("request_id")
	if requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "request_id is required",
		})
		return
	}

	detail, err := h.db.GetRequestDetail(requestID)
	if err != nil {
		logger.Error("Failed to get request detail",
			zap.String("request_id", requestID),
			zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Request not found",
		})
		return
	}

	// Возвращаем только тело запроса и метаданные
	c.JSON(http.StatusOK, gin.H{
		"request_id":         detail.RequestID,
		"timestamp":          detail.Timestamp,
		"method":             detail.Method,
		"path":               detail.Path,
		"content_type":       detail.ContentType,
		"body_size_bytes":    detail.BodySizeBytes,
		"body_text":          detail.BodyText,
		"has_sensitive_data": detail.HasSensitiveData,
		"headers":            detail.Headers,
	})
}

// GetErrorAnalytics возвращает аналитику по ошибкам запросов
func (h *RequestAnalysisHandler) GetErrorAnalytics(c *gin.Context) {
	periodStr := c.DefaultQuery("period", "24h")

	// Парсим период
	since := time.Now().Add(-24 * time.Hour)
	if periodStr != "" {
		duration, err := time.ParseDuration(periodStr)
		if err == nil {
			since = time.Now().Add(-duration)
		}
	}

	// Получаем все ошибки за период
	details, err := h.db.GetRequestDetailsByError(1000, since) // большой лимит для аналитики
	if err != nil {
		logger.Error("Failed to get error requests for analytics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve error analytics",
		})
		return
	}

	// Подготавливаем аналитику
	analytics := gin.H{
		"period":       periodStr,
		"since":        since,
		"total_errors": len(details),
	}

	if len(details) > 0 {
		// Статистика по категориям
		categoryStats := make(map[string]int)
		pathStats := make(map[string]int)
		statusStats := make(map[int]int)
		methodStats := make(map[string]int)
		hourlyStats := make(map[int]int)

		// Статистика по размеру тела запроса
		var totalBodySize int64
		var maxBodySize int64
		bodySizeRanges := map[string]int{
			"0-1KB":    0,
			"1-10KB":   0,
			"10-100KB": 0,
			"100KB+":   0,
		}

		for _, detail := range details {
			// Категории ошибок
			if detail.ErrorCategory != "" {
				categoryStats[detail.ErrorCategory]++
			}

			// Пути
			pathStats[detail.Path]++

			// HTTP статусы
			statusStats[detail.HTTPStatus]++

			// HTTP методы
			methodStats[detail.Method]++

			// Почасовая статистика
			hour := detail.Timestamp.Hour()
			hourlyStats[hour]++

			// Размеры тел запросов
			bodySize := detail.BodySizeBytes
			totalBodySize += bodySize
			if bodySize > maxBodySize {
				maxBodySize = bodySize
			}

			if bodySize == 0 {
				bodySizeRanges["0-1KB"]++
			} else if bodySize <= 1024 {
				bodySizeRanges["0-1KB"]++
			} else if bodySize <= 10*1024 {
				bodySizeRanges["1-10KB"]++
			} else if bodySize <= 100*1024 {
				bodySizeRanges["10-100KB"]++
			} else {
				bodySizeRanges["100KB+"]++
			}
		}

		avgBodySize := int64(0)
		if len(details) > 0 {
			avgBodySize = totalBodySize / int64(len(details))
		}

		analytics["by_category"] = categoryStats
		analytics["by_path"] = pathStats
		analytics["by_status_code"] = statusStats
		analytics["by_method"] = methodStats
		analytics["by_hour"] = hourlyStats
		analytics["body_size_stats"] = gin.H{
			"total_size_bytes":   totalBodySize,
			"max_size_bytes":     maxBodySize,
			"average_size_bytes": avgBodySize,
			"size_distribution":  bodySizeRanges,
		}

		// Топ-5 самых частых ошибок
		type errorFreq struct {
			Category string `json:"category"`
			Count    int    `json:"count"`
		}

		var topErrors []errorFreq
		for category, count := range categoryStats {
			topErrors = append(topErrors, errorFreq{
				Category: category,
				Count:    count,
			})
		}

		// Простая сортировка топ-5
		for i := 0; i < len(topErrors)-1; i++ {
			for j := i + 1; j < len(topErrors); j++ {
				if topErrors[i].Count < topErrors[j].Count {
					topErrors[i], topErrors[j] = topErrors[j], topErrors[i]
				}
			}
		}

		if len(topErrors) > 5 {
			topErrors = topErrors[:5]
		}

		analytics["top_error_categories"] = topErrors
	}

	c.JSON(http.StatusOK, analytics)
}
