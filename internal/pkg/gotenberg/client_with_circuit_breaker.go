package gotenberg

import (
	"context"
	"os"
	"time"

	"pdf-service-go/internal/pkg/circuitbreaker"
)



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
		// Сначала выполняем проверку здоровья
		if err := c.client.HealthCheck(); err != nil {
			return err
		}
		// Если проверка здоровья прошла успешно, выполняем конвертацию
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

// GetHandler возвращает обработчик статистики из базового клиента
func (c *ClientWithCircuitBreaker) GetHandler() (interface {
	TrackGotenbergRequest(duration time.Duration, hasError bool, isHealthCheck bool)
}, bool) {
	return c.client.GetHandler()
}

// SetHandler устанавливает обработчик статистики для базового клиента
func (c *ClientWithCircuitBreaker) SetHandler(handler interface {
	TrackGotenbergRequest(duration time.Duration, hasError bool, isHealthCheck bool)
}) {
	c.client.SetHandler(handler)
}
