package gotenberg

import (
	"context"
	"os"
	"strings"
	"time"

	"pdf-service-go/internal/pkg/circuitbreaker"
	"pdf-service-go/internal/pkg/logger"
	"pdf-service-go/internal/pkg/retry"
)

// ClientWithRetryAndCircuitBreaker комбинирует retry и circuit breaker механизмы
type ClientWithRetryAndCircuitBreaker struct {
	client  *Client
	cb      *circuitbreaker.CircuitBreaker
	retrier *retry.Retrier
}

// NewClientWithRetryAndCircuitBreaker создает нового клиента с retry и circuit breaker механизмами
func NewClientWithRetryAndCircuitBreaker(baseURL string) *ClientWithRetryAndCircuitBreaker {
	// Инициализируем логгер, если он еще не инициализирован
	if logger.Log == nil {
		if err := logger.Init("info"); err != nil {
			panic(err)
		}
	}

	client := NewClient(baseURL)

	// Создаем circuit breaker с оптимизированными настройками
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		Name:             "gotenberg",
		FailureThreshold: getEnvIntWithDefault("CIRCUIT_BREAKER_FAILURE_THRESHOLD", 10),
		ResetTimeout:     getEnvDurationWithDefault("CIRCUIT_BREAKER_RESET_TIMEOUT", 5*time.Second),
		HalfOpenMaxCalls: getEnvIntWithDefault("CIRCUIT_BREAKER_HALF_OPEN_MAX_CALLS", 5),
		SuccessThreshold: getEnvIntWithDefault("CIRCUIT_BREAKER_SUCCESS_THRESHOLD", 3),
		PodName:          os.Getenv("POD_NAME"),
		Namespace:        os.Getenv("POD_NAMESPACE"),
	})

	// Создаем retrier с оптимизированными настройками
	retrier := retry.New(
		"gotenberg",
		logger.Log,
		retry.WithMaxAttempts(3),
		retry.WithInitialDelay(20*time.Millisecond),
		retry.WithMaxDelay(500*time.Millisecond),
		retry.WithBackoffFactor(1.5),
	)

	return &ClientWithRetryAndCircuitBreaker{
		client:  client,
		cb:      cb,
		retrier: retrier,
	}
}

// ConvertDocxToPDF конвертирует DOCX в PDF с использованием retry и circuit breaker механизмов
func (c *ClientWithRetryAndCircuitBreaker) ConvertDocxToPDF(docxPath string) ([]byte, error) {
	var result []byte
	err := c.retrier.Do(context.Background(), func(ctx context.Context) error {
		// Проверяем состояние CB перед retry
		if c.cb.State() == circuitbreaker.StateOpen {
			return circuitbreaker.ErrCircuitOpen
		}

		// Выполняем операцию через circuit breaker
		return c.cb.Execute(ctx, func() error {
			// Сначала выполняем проверку здоровья без отслеживания в статистике
			if err := c.client.HealthCheck(true); err != nil {
				// Определяем тип ошибки и получаем соответствующую конфигурацию retry
				errorType := classifyError(err)
				config := retry.GetRetryConfig(errorType)

				// Обновляем настройки retry для текущей ошибки
				c.retrier.UpdateConfig(config)
				return err
			}

			// Если проверка здоровья прошла успешно, выполняем конвертацию
			var err error
			result, err = c.client.ConvertDocxToPDF(docxPath)
			if err != nil {
				// Также классифицируем ошибку конвертации
				errorType := classifyError(err)
				config := retry.GetRetryConfig(errorType)
				c.retrier.UpdateConfig(config)
			}
			return err
		})
	})
	return result, err
}

// classifyError определяет тип ошибки для выбора стратегии retry
func classifyError(err error) retry.ErrorType {
	if err == nil {
		return retry.ErrorTypeUnknown
	}

	switch {
	case isConnectionError(err):
		return retry.ErrorTypeConnection
	case isTimeout(err):
		return retry.ErrorTypeTimeout
	case isValidationError(err):
		return retry.ErrorTypeValidation
	default:
		return retry.ErrorTypeUnknown
	}
}

// isConnectionError проверяет, является ли ошибка ошибкой соединения
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "connection reset by peer")
}

// isTimeout проверяет, является ли ошибка таймаутом
func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "context deadline exceeded")
}

// isValidationError проверяет, является ли ошибка ошибкой валидации
func isValidationError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "validation failed") ||
		strings.Contains(errStr, "invalid input") ||
		strings.Contains(errStr, "bad request") ||
		strings.Contains(errStr, "400")
}

// State возвращает текущее состояние Circuit Breaker
func (c *ClientWithRetryAndCircuitBreaker) State() circuitbreaker.State {
	return c.cb.State()
}

// IsHealthy возвращает true, если Circuit Breaker в здоровом состоянии
func (c *ClientWithRetryAndCircuitBreaker) IsHealthy() bool {
	return c.cb.IsHealthy()
}

// GetHandler возвращает обработчик статистики из базового клиента
func (c *ClientWithRetryAndCircuitBreaker) GetHandler() (interface {
	TrackGotenbergRequest(duration time.Duration, hasError bool, isHealthCheck bool)
}, bool) {
	return c.client.GetHandler()
}

// SetHandler устанавливает обработчик статистики для базового клиента
func (c *ClientWithRetryAndCircuitBreaker) SetHandler(handler interface {
	TrackGotenbergRequest(duration time.Duration, hasError bool, isHealthCheck bool)
}) {
	c.client.SetHandler(handler)
}
