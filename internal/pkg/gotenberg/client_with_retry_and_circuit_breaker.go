package gotenberg

import (
	"context"
	"os"
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
		retry.WithMaxAttempts(getEnvIntWithDefault("GOTENBERG_RETRY_MAX_ATTEMPTS", 5)),
		retry.WithInitialDelay(getEnvDurationWithDefault("GOTENBERG_RETRY_INITIAL_DELAY", 50*time.Millisecond)),
		retry.WithMaxDelay(getEnvDurationWithDefault("GOTENBERG_RETRY_MAX_DELAY", 1*time.Second)),
		retry.WithBackoffFactor(float64(getEnvIntWithDefault("GOTENBERG_RETRY_BACKOFF_FACTOR", 2))),
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
		return c.cb.Execute(ctx, func() error {
			// Сначала выполняем проверку здоровья без отслеживания в статистике
			if err := c.client.HealthCheck(true); err != nil {
				return err
			}
			// Если проверка здоровья прошла успешно, выполняем конвертацию
			var err error
			result, err = c.client.ConvertDocxToPDF(docxPath)
			return err
		})
	})
	return result, err
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
