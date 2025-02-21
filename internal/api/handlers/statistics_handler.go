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
	period := c.DefaultQuery("period", "all")

	// Добавляем отладочный вывод
	fmt.Printf("\n=== API Statistics Handler ===\n")
	fmt.Printf("Received request with period: %s\n", period)
	fmt.Printf("Request headers: %v\n", c.Request.Header)

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
		fmt.Printf("Invalid period: %s\n", period)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid period: %s", period)})
		return
	}

	stats, err := h.stats.GetStatisticsForPeriod(period)
	if err != nil {
		fmt.Printf("Error getting statistics: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Добавляем отладочный вывод результатов
	fmt.Printf("\n=== API Response Summary ===\n")
	fmt.Printf("Total requests: %d\n", stats.Requests.Total)
	fmt.Printf("Success requests: %d\n", stats.Requests.Success)
	fmt.Printf("Failed requests: %d\n", stats.Requests.Failed)
	fmt.Printf("By day of week: %v\n", stats.Requests.ByDayOfWeek)
	fmt.Printf("By hour of day: %v\n", stats.Requests.ByHourOfDay)
	fmt.Printf("=== End of API Handler ===\n\n")

	c.JSON(http.StatusOK, stats)
}
