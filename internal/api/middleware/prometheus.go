package middleware

import (
	"pdf-service-go/internal/pkg/metrics"
	"time"

	"github.com/gin-gonic/gin"
)

// PrometheusMiddleware middleware для сбора метрик HTTP запросов
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Продолжаем обработку запроса
		c.Next()

		// Собираем метрики после обработки
		duration := time.Since(start).Seconds()
		status := c.Writer.Status()
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		method := c.Request.Method

		// Увеличиваем счетчик запросов
		metrics.HTTPRequestsTotal.WithLabelValues(method, path, string(rune(status))).Inc()

		// Записываем длительность запроса
		metrics.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
	}
}
