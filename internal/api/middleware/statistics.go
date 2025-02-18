package middleware

import (
	"pdf-service-go/internal/pkg/statistics"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// StatisticsMiddleware middleware для сбора статистики
func StatisticsMiddleware() gin.HandlerFunc {
	stats := statistics.GetInstance()

	return func(c *gin.Context) {
		// Пропускаем эндпоинты health check, metrics и статистику
		path := c.Request.URL.Path
		if path == "/health" ||
			path == "/metrics" ||
			path == "/stats" ||
			path == "/api/v1/statistics" ||
			path == "/statistics" ||
			strings.HasPrefix(path, "/static/") {
			c.Next()
			return
		}

		start := time.Now()

		// Выполняем запрос
		c.Next()

		// Вычисляем длительность
		duration := time.Since(start)

		// Определяем успешность запроса
		success := c.Writer.Status() < 400

		// Обновляем статистику
		stats.TrackRequest(duration, success)
	}
}
