package errortracker

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"pdf-service-go/internal/pkg/statistics"
	"pdf-service-go/internal/pkg/tracing"

	"go.opentelemetry.io/otel/trace"
)

// ErrorTracker отслеживает и записывает ошибки с контекстом
type ErrorTracker struct {
	stats *statistics.Statistics
}

// NewErrorTracker создает новый трекер ошибок
func NewErrorTracker() *ErrorTracker {
	return &ErrorTracker{
		stats: statistics.GetInstance(),
	}
}

// TrackError записывает ошибку с полным контекстом
func (et *ErrorTracker) TrackError(ctx context.Context, err error, opts ...ErrorOption) {
	if err == nil || et.stats == nil {
		return
	}

	// Создаем базовую структуру ошибки
	errorDetails := &statistics.ErrorDetails{
		Timestamp:      time.Now(),
		Message:        err.Error(),
		RequestDetails: make(map[string]interface{}),
	}

	// Извлекаем информацию из контекста
	et.extractContextInfo(ctx, errorDetails)

	// Получаем стек-трейс
	errorDetails.StackTrace = getStackTrace(3) // Пропускаем 3 фрейма

	// Применяем опции
	for _, opt := range opts {
		opt(errorDetails)
	}

	// Автоматически определяем тип и компонент если не заданы
	if errorDetails.ErrorType == "" {
		errorDetails.ErrorType = statistics.ClassifyError(errorDetails.Message)
	}

	if errorDetails.Component == "" {
		errorDetails.Component = statistics.DetermineComponent(errorDetails.StackTrace, errorDetails.Message)
	}

	if errorDetails.Severity == "" {
		errorDetails.Severity = statistics.DetermineSeverity(errorDetails.ErrorType, errorDetails.HTTPStatus)
	}

	// Пороговые алерты по stage-timing (если в контексте есть timings файл — попробуем прочитать)
	// Лёгкая эвристика: не читаем файл, а ожидаем, что upstream уже добавит атрибуты при необходимости.
	// Здесь добавим только маркеры в RequestDetails для визуализации.
	if v := getFromContext(ctx, "stage_name"); v != "" {
		if errorDetails.RequestDetails == nil {
			errorDetails.RequestDetails = map[string]interface{}{}
		}
		errorDetails.RequestDetails["stage"] = v
	}

	// Записываем в базу данных
	if logErr := et.stats.LogError(errorDetails); logErr != nil {
		// Если не удалось записать в БД, хотя бы логируем в консоль
		fmt.Printf("Failed to log error to database: %v\n", logErr)
	}

	// Записываем в трейсинг если доступен
	tracing.RecordError(ctx, err)
}

// extractContextInfo извлекает информацию из контекста
func (et *ErrorTracker) extractContextInfo(ctx context.Context, details *statistics.ErrorDetails) {
	// Извлекаем trace и span ID
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		details.TraceID = span.SpanContext().TraceID().String()
		details.SpanID = span.SpanContext().SpanID().String()
	}

	// Извлекаем HTTP информацию из контекста если доступна
	if requestID := getFromContext(ctx, "request_id"); requestID != "" {
		details.RequestID = requestID
	}

	if clientIP := getFromContext(ctx, "client_ip"); clientIP != "" {
		details.ClientIP = clientIP
	}

	if userAgent := getFromContext(ctx, "user_agent"); userAgent != "" {
		details.UserAgent = userAgent
	}

	if method := getFromContext(ctx, "http_method"); method != "" {
		details.HTTPMethod = method
	}

	if path := getFromContext(ctx, "http_path"); path != "" {
		details.HTTPPath = path
	}

	// Дополнительная информация
	details.RequestDetails["timestamp"] = time.Now().Format(time.RFC3339)
	details.RequestDetails["goroutine_id"] = getGoroutineID()

	// Если путь к сохраненному payload присутствует в контексте — добавим ссылку
	if v := getFromContext(ctx, "request_body_file_path"); v != "" {
		details.RequestDetails["request_payload_path"] = v
	}
}

// ErrorOption функция для настройки деталей ошибки
type ErrorOption func(*statistics.ErrorDetails)

// WithComponent устанавливает компонент
func WithComponent(component string) ErrorOption {
	return func(e *statistics.ErrorDetails) {
		e.Component = component
	}
}

// WithErrorType устанавливает тип ошибки
func WithErrorType(errorType string) ErrorOption {
	return func(e *statistics.ErrorDetails) {
		e.ErrorType = errorType
	}
}

// WithSeverity устанавливает критичность
func WithSeverity(severity string) ErrorOption {
	return func(e *statistics.ErrorDetails) {
		e.Severity = severity
	}
}

// WithHTTPStatus устанавливает HTTP статус
func WithHTTPStatus(status int) ErrorOption {
	return func(e *statistics.ErrorDetails) {
		e.HTTPStatus = status
	}
}

// WithDuration устанавливает длительность операции
func WithDuration(duration time.Duration) ErrorOption {
	return func(e *statistics.ErrorDetails) {
		e.Duration = duration
	}
}

// WithRequestDetails добавляет дополнительные детали запроса
func WithRequestDetails(key string, value interface{}) ErrorOption {
	return func(e *statistics.ErrorDetails) {
		if e.RequestDetails == nil {
			e.RequestDetails = make(map[string]interface{})
		}
		e.RequestDetails[key] = value
	}
}

// WithRequestID устанавливает ID запроса
func WithRequestID(requestID string) ErrorOption {
	return func(e *statistics.ErrorDetails) {
		e.RequestID = requestID
	}
}

// Вспомогательные функции
func getStackTrace(skip int) string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

func getFromContext(ctx context.Context, key string) string {
	if value := ctx.Value(key); value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func getGoroutineID() string {
	buf := make([]byte, 64)
	buf = buf[:runtime.Stack(buf, false)]
	// Простое извлечение ID горутины из стека
	for i := 0; i < len(buf); i++ {
		if buf[i] == ' ' {
			return string(buf[10:i])
		}
	}
	return "unknown"
}

// Глобальная переменная для удобства
var DefaultTracker = NewErrorTracker()

// TrackError глобальная функция для записи ошибок
func TrackError(ctx context.Context, err error, opts ...ErrorOption) {
	DefaultTracker.TrackError(ctx, err, opts...)
}
