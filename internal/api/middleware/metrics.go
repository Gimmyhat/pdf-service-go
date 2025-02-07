package middleware

import (
	"pdf-service-go/internal/metrics"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// PrometheusMiddleware собирает метрики для каждого запроса
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		operation := c.Request.URL.Path

		// Вызываем следующий обработчик
		c.Next()

		// Увеличиваем счетчик запросов
		status := strconv.Itoa(c.Writer.Status())
		metrics.RequestsTotal.WithLabelValues(status, operation).Inc()

		// Записываем длительность запроса
		duration := time.Since(start).Seconds()
		metrics.RequestDuration.WithLabelValues(operation).Observe(duration)

		// Если была ошибка, увеличиваем счетчик ошибок
		if c.Writer.Status() >= 400 {
			errorType := "server_error"
			if c.Writer.Status() < 500 {
				errorType = "client_error"
			}
			metrics.ErrorsTotal.WithLabelValues(errorType, operation).Inc()
		}
	}
}
