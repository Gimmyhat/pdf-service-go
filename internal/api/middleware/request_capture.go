package middleware

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"pdf-service-go/internal/pkg/logger"
	"pdf-service-go/internal/pkg/statistics"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestCaptureMiddleware создает middleware для захвата детальной информации о запросах
func RequestCaptureMiddleware(db *statistics.PostgresDB, config statistics.RequestCaptureConfig) gin.HandlerFunc {
	// Компилируем регулярные выражения для чувствительных данных
	sensitivePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(password|token|secret|key|auth)`),
		regexp.MustCompile(`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`),          // Credit card numbers
		regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`), // Email addresses
	}

	return func(c *gin.Context) {
		// Архивируем только запросы конвертации JSON→PDF
		if !isConversionRequestPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Дополнительно проверяем пользовательские исключения
		if !shouldCaptureRequest(c.Request.URL.Path, config.ExcludePaths) {
			c.Next()
			return
		}

		// Генерируем уникальный ID запроса
		requestID := generateRequestID()
		c.Set("request_id", requestID)
		// Дублируем в заголовок ответа для корреляции на клиенте
		c.Writer.Header().Set("X-Request-ID", requestID)

		// Захватываем данные запроса
		capture := &statistics.RequestCapture{
			RequestID:   requestID,
			Method:      c.Request.Method,
			Path:        c.Request.URL.Path,
			ClientIP:    getClientIP(c),
			UserAgent:   c.Request.UserAgent(),
			Headers:     extractHeaders(c.Request.Header, config.ExcludeHeaders),
			ContentType: c.Request.Header.Get("Content-Type"),
			StartTime:   time.Now(),
		}

		// Захватываем body запроса если нужно
		if shouldCaptureBody(c.Request.Method) && c.Request.Body != nil {
			bodyBytes, err := captureRequestBody(c, config.MaxBodySize)
			if err != nil {
				logger.Error("Failed to capture request body", zap.Error(err))
			} else {
				capture.Body = bodyBytes
			}
		}

		// Пробросим ключевые метаданные в request.Context, чтобы downstream мог их читать
		{
			ctx := context.WithValue(c.Request.Context(), "request_id", requestID)
			ctx = context.WithValue(ctx, "client_ip", capture.ClientIP)
			ctx = context.WithValue(ctx, "user_agent", capture.UserAgent)
			ctx = context.WithValue(ctx, "http_method", capture.Method)
			ctx = context.WithValue(ctx, "http_path", capture.Path)
			c.Request = c.Request.WithContext(ctx)
		}

		// Сохраним тело запроса в файл сразу (до выполнения обработки), чтобы не зависеть от контекста позднее
		var requestBodyFilePath string
		if len(capture.Body) > 0 {
			if path, err := saveRequestBodyToFile(capture.RequestID, capture.Body); err == nil {
				requestBodyFilePath = path
			} else {
				logger.Warn("Failed to persist request body to file", zap.Error(err))
			}
		}

		// Сохраним путь к файлу в Gin context и пробросим в request.Context для последующих обработчиков/трекинга
		if requestBodyFilePath != "" {
			c.Set("request_body_file_path", requestBodyFilePath)
			ctx := context.WithValue(c.Request.Context(), "request_body_file_path", requestBodyFilePath)
			c.Request = c.Request.WithContext(ctx)
		}

		// Выполняем обработку запроса
		c.Next()

		// Сохраняем детальную информацию только если включен захват
		if config.EnableCapture {
			// Снимем необходимые значения из контекста ДО запуска горутины
			statusCode := c.Writer.Status()
			duration := time.Since(capture.StartTime)

			go func(status int, dur time.Duration, bodyPath string) {
				// Получаем актуальный DB-инстанс динамически (мог инициализироваться после старта)
				activeDB := statistics.GetPostgresDB()
				if activeDB == nil {
					logger.Warn("Request capture skipped: statistics DB is not initialized yet",
						zap.String("path", capture.Path),
						zap.String("method", capture.Method),
					)
					return
				}
				if err := saveRequestDetailSnapshot(activeDB, capture, status, dur, bodyPath, sensitivePatterns, config); err != nil {
					logger.Error("Failed to save request detail", zap.Error(err))
				}
			}(statusCode, duration, requestBodyFilePath)
		}
	}
}

// shouldCaptureRequest проверяет, нужно ли захватывать данный запрос
func shouldCaptureRequest(path string, excludePaths []string) bool {
	// Исключаем системные пути
	systemPaths := []string{"/health", "/metrics", "/favicon.ico"}
	for _, systemPath := range systemPaths {
		if path == systemPath {
			return false
		}
	}

	// Проверяем пользовательские исключения
	for _, excludePath := range excludePaths {
		if strings.HasPrefix(path, excludePath) {
			return false
		}
	}

	return true
}

// isConversionRequestPath возвращает true, если путь относится к конвертации JSON→PDF
func isConversionRequestPath(path string) bool {
	if path == "/api/v1/docx" || path == "/generate-pdf" {
		return true
	}
	return false
}

// shouldCaptureBody проверяет, нужно ли захватывать body запроса
func shouldCaptureBody(method string) bool {
	return method == "POST" || method == "PUT" || method == "PATCH"
}

// captureRequestBody захватывает body запроса с ограничением размера
func captureRequestBody(c *gin.Context, maxSize int64) ([]byte, error) {
	if c.Request.Body == nil {
		return nil, nil
	}

	// Читаем body с ограничением размера
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxSize))
	if err != nil {
		return nil, err
	}

	// Восстанавливаем body для дальнейшей обработки
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	return body, nil
}

// extractHeaders извлекает заголовки, исключая чувствительные
func extractHeaders(headers http.Header, excludeHeaders []string) map[string]string {
	result := make(map[string]string)

	// Системные заголовки для исключения
	systemExcludes := []string{"authorization", "cookie", "x-api-key", "x-auth-token"}

	for name, values := range headers {
		lowerName := strings.ToLower(name)

		// Проверяем системные исключения
		excluded := false
		for _, exclude := range systemExcludes {
			if lowerName == exclude {
				excluded = true
				break
			}
		}

		// Проверяем пользовательские исключения
		if !excluded {
			for _, exclude := range excludeHeaders {
				if strings.ToLower(exclude) == lowerName {
					excluded = true
					break
				}
			}
		}

		if !excluded && len(values) > 0 {
			result[name] = values[0]
		}
	}

	return result
}

// getClientIP получает IP адрес клиента
func getClientIP(c *gin.Context) string {
	// Проверяем заголовки прокси
	if ip := c.Request.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := c.Request.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return c.ClientIP()
}

// generateRequestID генерирует уникальный ID запроса
func generateRequestID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("req_%s_%d", hex.EncodeToString(bytes), time.Now().Unix())
}

// saveRequestDetail сохраняет детальную информацию о запросе
func saveRequestDetail(db *statistics.PostgresDB, capture *statistics.RequestCapture, c *gin.Context, sensitivePatterns []*regexp.Regexp, config statistics.RequestCaptureConfig) {
	// Получаем информацию о результате обработки
	success := c.Writer.Status() < 400
	duration := time.Since(capture.StartTime)

	// Проверяем, нужно ли сохранять (только ошибки или все)
	if config.CaptureOnlyErrors && success {
		return
	}

	// Подготавливаем текст body
	bodyText := ""
	hasSensitiveData := false
	if len(capture.Body) > 0 {
		bodyText = string(capture.Body)

		// Проверяем на чувствительные данные
		if config.MaskSensitiveData {
			for _, pattern := range sensitivePatterns {
				if pattern.MatchString(bodyText) {
					hasSensitiveData = true
					bodyText = pattern.ReplaceAllString(bodyText, "[MASKED]")
				}
			}
		}
	}

	// Определяем категорию ошибки
	errorCategory := ""
	if !success {
		errorCategory = categorizeError(c.Writer.Status(), duration, capture.Path)
	}

	// Создаем запись для сохранения
	// Попробуем получить пути к файлам из контекста (если сохранены)
	var requestFilePathPtr *string
	if v, exists := c.Get("request_body_file_path"); exists {
		if s, ok := v.(string); ok && s != "" {
			requestFilePathPtr = &s
		}
	}

	detail := &statistics.RequestDetail{
		RequestID:        capture.RequestID,
		Timestamp:        capture.StartTime,
		Method:           capture.Method,
		Path:             capture.Path,
		ClientIP:         capture.ClientIP,
		UserAgent:        capture.UserAgent,
		Headers:          capture.Headers,
		BodyText:         bodyText,
		BodySizeBytes:    int64(len(capture.Body)),
		Success:          success,
		HTTPStatus:       c.Writer.Status(),
		DurationNs:       duration.Nanoseconds(),
		ContentType:      capture.ContentType,
		HasSensitiveData: hasSensitiveData,
		ErrorCategory:    errorCategory,
		RequestFilePath:  requestFilePathPtr,
	}

	// Сохраняем в базу данных
	if err := db.SaveRequestDetail(detail); err != nil {
		logger.Error("Failed to save request detail",
			zap.String("request_id", capture.RequestID),
			zap.Error(err))
	}
}

// saveRequestDetailSnapshot сохраняет детальную информацию, не обращаясь к контексту Gin после завершения запроса
func saveRequestDetailSnapshot(db *statistics.PostgresDB, capture *statistics.RequestCapture, status int, duration time.Duration, bodyFilePath string, sensitivePatterns []*regexp.Regexp, config statistics.RequestCaptureConfig) error {
	success := status < 400
	if config.CaptureOnlyErrors && success {
		return nil
	}

	bodyText := ""
	hasSensitiveData := false
	if len(capture.Body) > 0 {
		bodyText = string(capture.Body)
		if config.MaskSensitiveData {
			for _, pattern := range sensitivePatterns {
				if pattern.MatchString(bodyText) {
					hasSensitiveData = true
					bodyText = pattern.ReplaceAllString(bodyText, "[MASKED]")
				}
			}
		}
	}

	errorCategory := ""
	if !success {
		errorCategory = categorizeError(status, duration, capture.Path)
	}

	var requestFilePathPtr *string
	if bodyFilePath != "" {
		requestFilePathPtr = &bodyFilePath
	}

	detail := &statistics.RequestDetail{
		RequestID:        capture.RequestID,
		Timestamp:        capture.StartTime,
		Method:           capture.Method,
		Path:             capture.Path,
		ClientIP:         capture.ClientIP,
		UserAgent:        capture.UserAgent,
		Headers:          capture.Headers,
		BodyText:         bodyText,
		BodySizeBytes:    int64(len(capture.Body)),
		Success:          success,
		HTTPStatus:       status,
		DurationNs:       duration.Nanoseconds(),
		ContentType:      capture.ContentType,
		HasSensitiveData: hasSensitiveData,
		ErrorCategory:    errorCategory,
		RequestFilePath:  requestFilePathPtr,
	}

	return db.SaveRequestDetail(detail)
}

// saveRequestBodyToFile сохраняет тело запроса на диск и возвращает путь к файлу
func saveRequestBodyToFile(requestID string, body []byte) (string, error) {
	baseDir := getArtifactsBaseDir()
	reqDir := filepath.Join(baseDir, "requests")
	if err := os.MkdirAll(reqDir, 0o755); err != nil {
		return "", err
	}
	filename := filepath.Join(reqDir, fmt.Sprintf("%s.json", requestID))
	if err := os.WriteFile(filename, body, 0o644); err != nil {
		return "", err
	}
	return filename, nil
}

// getArtifactsBaseDir возвращает базовую директорию для артефактов
func getArtifactsBaseDir() string {
	if v := os.Getenv("ARTIFACTS_DIR"); v != "" {
		return v
	}
	return "/app/data/artifacts"
}

// categorizeError определяет категорию ошибки
func categorizeError(statusCode int, duration time.Duration, path string) string {
	if statusCode >= 500 {
		if duration > 60*time.Second {
			return "timeout_error"
		} else if duration < 100*time.Millisecond {
			return "instant_failure"
		}
		return "server_error"
	} else if statusCode >= 400 {
		if statusCode == 422 {
			return "validation_error"
		} else if statusCode == 404 {
			return "not_found"
		}
		return "client_error"
	}
	return ""
}
