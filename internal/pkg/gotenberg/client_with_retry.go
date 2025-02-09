package gotenberg

import (
	"context"
	"time"

	"pdf-service-go/internal/pkg/logger"
	"pdf-service-go/internal/pkg/retry"
)

// ClientWithRetry добавляет retry механизм к клиенту Gotenberg
type ClientWithRetry struct {
	client  *Client
	retrier *retry.Retrier
}

// NewClientWithRetry создает нового клиента с retry механизмом
func NewClientWithRetry(baseURL string) *ClientWithRetry {
	client := NewClient(baseURL)

	// Создаем retrier с конфигурацией из переменных окружения
	retrier := retry.New(
		"gotenberg",
		logger.Log,
		retry.WithMaxAttempts(getEnvIntWithDefault("GOTENBERG_RETRY_MAX_ATTEMPTS", 3)),
		retry.WithInitialDelay(getEnvDurationWithDefault("GOTENBERG_RETRY_INITIAL_DELAY", 100*time.Millisecond)),
		retry.WithMaxDelay(getEnvDurationWithDefault("GOTENBERG_RETRY_MAX_DELAY", 2*time.Second)),
		retry.WithBackoffFactor(float64(getEnvIntWithDefault("GOTENBERG_RETRY_BACKOFF_FACTOR", 2))),
	)

	return &ClientWithRetry{
		client:  client,
		retrier: retrier,
	}
}

// ConvertDocxToPDF конвертирует DOCX в PDF с использованием retry механизма
func (c *ClientWithRetry) ConvertDocxToPDF(docxPath string) ([]byte, error) {
	var result []byte
	err := c.retrier.Do(context.Background(), func(ctx context.Context) error {
		var err error
		result, err = c.client.ConvertDocxToPDF(docxPath)
		return err
	})
	return result, err
}
