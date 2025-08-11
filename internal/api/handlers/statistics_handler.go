package handlers

import (
	"fmt"
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
	// Лениво получаем актуальный инстанс (мог появиться позже из-за ретраев подключения)
	if h.stats == nil {
		h.stats = statistics.GetInstance()
	}
	if h.stats == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "statistics subsystem is not ready"})
		return
	}
	period := c.DefaultQuery("period", "all")

	// Проверяем допустимость периода
	validPeriods := map[string]bool{
		"15min":   true,
		"1hour":   true,
		"5hours":  true,
		"24hours": true,
		"week":    true,
		"month":   true,
		"all":     true,
		"":        true,
	}

	if !validPeriods[period] {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid period: %s", period)})
		return
	}

	stats, err := h.stats.GetStatisticsForPeriod(period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
