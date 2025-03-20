package middleware

import (
	"pdf-service-go/internal/pkg/logger"
	"pdf-service-go/internal/pkg/statistics"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// StatisticsMiddleware middleware для сбора статистики
func StatisticsMiddleware() gin.HandlerFunc {
	stats := statistics.GetInstance()

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method

		// Отслеживаем только запросы на генерацию файлов
		if (path == "/api/v1/docx" || path == "/generate-pdf") && method == "POST" {
			start := time.Now()
			c.Next()
			duration := time.Since(start)

			// Определяем успешность запроса (2xx или 3xx)
			status := c.Writer.Status()
			success := status >= 200 && status < 400

			// Добавляем отладочное логирование
			logger.Debug("Request statistics",
				zap.String("path", path),
				zap.String("method", method),
				zap.Int("status", status),
				zap.Bool("success", success),
				zap.Duration("duration", duration),
			)

			// Обновляем статистику
			stats.TrackRequest(path, method, duration, success)
		} else {
			c.Next()
		}
	}
}
