package statistics

import (
	"strings"
	"time"
)

// ErrorDetails представляет детальную информацию об ошибке
type ErrorDetails struct {
	ID             int64                  `json:"id" db:"id"`
	Timestamp      time.Time              `json:"timestamp" db:"timestamp"`
	RequestID      string                 `json:"request_id" db:"request_id"`
	TraceID        string                 `json:"trace_id" db:"trace_id"`
	SpanID         string                 `json:"span_id" db:"span_id"`
	ErrorType      string                 `json:"error_type" db:"error_type"`
	Component      string                 `json:"component" db:"component"`
	Message        string                 `json:"message" db:"message"`
	StackTrace     string                 `json:"stack_trace,omitempty" db:"stack_trace"`
	RequestDetails map[string]interface{} `json:"request_details" db:"request_details"`
	ClientIP       string                 `json:"client_ip" db:"client_ip"`
	UserAgent      string                 `json:"user_agent" db:"user_agent"`
	HTTPMethod     string                 `json:"http_method" db:"http_method"`
	HTTPPath       string                 `json:"http_path" db:"http_path"`
	HTTPStatus     int                    `json:"http_status" db:"http_status"`
	Duration       time.Duration          `json:"duration_ms" db:"duration_ns"`
	Severity       string                 `json:"severity" db:"severity"`
}

// ErrorPattern представляет паттерн ошибок для анализа
type ErrorPattern struct {
	ErrorType   string        `json:"error_type"`
	Component   string        `json:"component"`
	Count       int64         `json:"count"`
	Frequency   string        `json:"frequency"`
	Trend       string        `json:"trend"`
	LastOccured time.Time     `json:"last_occurred"`
	AvgDuration time.Duration `json:"avg_duration_ms"`
}

// ErrorSummary представляет сводку ошибок
type ErrorSummary struct {
	RecentErrors   []ErrorDetails `json:"recent_errors"`
	ErrorPatterns  []ErrorPattern `json:"error_patterns"`
	TotalErrors    int64          `json:"total_errors"`
	ErrorsLast24h  int64          `json:"errors_last_24h"`
	ErrorsLastHour int64          `json:"errors_last_hour"`
	TopErrorTypes  []ErrorPattern `json:"top_error_types"`
}

// GetErrorSolutions возвращает автоматические решения для ошибки
func GetErrorSolutions(errorType, component string) []string {
	solutions := map[string][]string{
		"timeout": {
			"Проверить ресурсы Gotenberg (CPU/Memory)",
			"Увеличить timeout в конфигурации до 90-120s",
			"Проверить размер и сложность документа",
			"Проверить загрузку сети между сервисами",
		},
		"validation": {
			"Проверить формат входных данных",
			"Убедиться в наличии всех обязательных полей",
			"Проверить размеры загружаемых файлов",
			"Валидировать JSON схему запроса",
		},
		"panic": {
			"Проверить логи для стек-трейса",
			"Перезапустить сервис если проблема критична",
			"Проверить входные данные на некорректные значения",
			"Обратиться к разработчикам",
		},
		"connection": {
			"Проверить доступность Gotenberg сервиса",
			"Проверить DNS резолюцию",
			"Проверить сетевые политики Kubernetes",
			"Перезапустить Gotenberg если необходимо",
		},
		"resource": {
			"Увеличить лимиты памяти для подов",
			"Проверить дисковое пространство",
			"Очистить временные файлы",
			"Масштабировать количество реплик",
		},
	}

	if sols, exists := solutions[errorType]; exists {
		return sols
	}

	// Общие решения
	return []string{
		"Проверить логи сервиса для дополнительной информации",
		"Проверить статус всех компонентов системы",
		"Обратиться к документации или поддержке",
	}
}

// ClassifyError определяет тип ошибки по сообщению
func ClassifyError(message string) string {
	timeoutKeywords := []string{"timeout", "deadline exceeded", "context canceled"}
	validationKeywords := []string{"validation", "invalid", "bad request", "missing"}
	connectionKeywords := []string{"connection", "dial", "network", "refused"}
	resourceKeywords := []string{"memory", "disk", "space", "limit"}

	msgLower := strings.ToLower(message)

	for _, keyword := range timeoutKeywords {
		if strings.Contains(msgLower, keyword) {
			return "timeout"
		}
	}

	for _, keyword := range validationKeywords {
		if strings.Contains(msgLower, keyword) {
			return "validation"
		}
	}

	for _, keyword := range connectionKeywords {
		if strings.Contains(msgLower, keyword) {
			return "connection"
		}
	}

	for _, keyword := range resourceKeywords {
		if strings.Contains(msgLower, keyword) {
			return "resource"
		}
	}

	return "unknown"
}

// DetermineComponent определяет компонент по стек-трейсу или сообщению
func DetermineComponent(stackTrace, message string) string {
	text := strings.ToLower(stackTrace + " " + message)

	if strings.Contains(text, "gotenberg") {
		return "gotenberg"
	}
	if strings.Contains(text, "docx") || strings.Contains(text, "generate") {
		return "docx"
	}
	if strings.Contains(text, "pdf") {
		return "pdf"
	}
	if strings.Contains(text, "postgres") || strings.Contains(text, "database") {
		return "database"
	}
	if strings.Contains(text, "api") || strings.Contains(text, "handler") {
		return "api"
	}

	return "system"
}

// DetermineSeverity определяет критичность ошибки
func DetermineSeverity(errorType string, httpStatus int) string {
	if httpStatus >= 500 {
		return "critical"
	}
	if errorType == "panic" {
		return "critical"
	}
	if errorType == "timeout" && httpStatus == 0 {
		return "high"
	}
	if httpStatus >= 400 && httpStatus < 500 {
		return "medium"
	}
	return "low"
}
