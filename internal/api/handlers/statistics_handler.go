package handlers

import (
	"net/http"
	"pdf-service-go/internal/pkg/statistics"

	"github.com/gin-gonic/gin"
)

// StatisticsHandler обработчик для статистики
type StatisticsHandler struct {
	stats *statistics.Statistics
}

// NewStatisticsHandler создает новый обработчик статистики
func NewStatisticsHandler() *StatisticsHandler {
	return &StatisticsHandler{
		stats: statistics.GetInstance(),
	}
}

// GetStatistics возвращает текущую статистику
func (h *StatisticsHandler) GetStatistics(c *gin.Context) {
	stats := h.stats.GetStatistics()
	c.JSON(http.StatusOK, stats)
}
