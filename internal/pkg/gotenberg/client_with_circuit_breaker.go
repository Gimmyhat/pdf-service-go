package gotenberg

import (
	"context"
	"os"
	"strconv"
	"time"

	"pdf-service-go/internal/pkg/circuitbreaker"
)

// getEnvIntWithDefault возвращает целочисленное значение переменной окружения или значение по умолчанию
func getEnvIntWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvDurationWithDefault возвращает значение длительности из переменной окружения или значение по умолчанию
func getEnvDurationWithDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// ClientWithCircuitBreaker добавляет Circuit Breaker к клиенту Gotenberg
type ClientWithCircuitBreaker struct {
	client *Client
	cb     *circuitbreaker.CircuitBreaker
}

// NewClientWithCircuitBreaker создает нового клиента с Circuit Breaker
func NewClientWithCircuitBreaker(baseURL string) *ClientWithCircuitBreaker {
	client := NewClient(baseURL)
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		Name:             "gotenberg",
		FailureThreshold: getEnvIntWithDefault("CIRCUIT_BREAKER_FAILURE_THRESHOLD", 5),
		ResetTimeout:     getEnvDurationWithDefault("CIRCUIT_BREAKER_RESET_TIMEOUT", 10*time.Second),
		HalfOpenMaxCalls: getEnvIntWithDefault("CIRCUIT_BREAKER_HALF_OPEN_MAX_CALLS", 2),
		SuccessThreshold: getEnvIntWithDefault("CIRCUIT_BREAKER_SUCCESS_THRESHOLD", 2),
		PodName:          os.Getenv("POD_NAME"),
		Namespace:        os.Getenv("POD_NAMESPACE"),
	})

	return &ClientWithCircuitBreaker{
		client: client,
		cb:     cb,
	}
}

// ConvertDocxToPDF конвертирует DOCX в PDF с использованием Circuit Breaker
func (c *ClientWithCircuitBreaker) ConvertDocxToPDF(docxPath string) ([]byte, error) {
	var result []byte
	err := c.cb.Execute(context.Background(), func() error {
		var err error
		result, err = c.client.ConvertDocxToPDF(docxPath)
		return err
	})
	return result, err
}

// State возвращает текущее состояние Circuit Breaker
func (c *ClientWithCircuitBreaker) State() circuitbreaker.State {
	return c.cb.State()
}

// IsHealthy возвращает true, если Circuit Breaker в здоровом состоянии
func (c *ClientWithCircuitBreaker) IsHealthy() bool {
	return c.cb.IsHealthy()
}
