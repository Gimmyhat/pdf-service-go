package retry

import (
	"errors"
	"fmt"
)

var (
	// ErrMaxAttemptsReached возникает когда исчерпаны все попытки
	ErrMaxAttemptsReached = errors.New("max retry attempts reached")
	// ErrInvalidConfig возникает при некорректной конфигурации
	ErrInvalidConfig = errors.New("invalid retry configuration")
)

// RetryError содержит информацию об ошибке retry
type RetryError struct {
	// Attempt номер попытки, на которой произошла ошибка
	Attempt int
	// OriginalError исходная ошибка
	OriginalError error
}

func (e *RetryError) Error() string {
	return fmt.Sprintf("retry attempt %d failed: %v", e.Attempt, e.OriginalError)
}

// Unwrap возвращает оригинальную ошибку
func (e *RetryError) Unwrap() error {
	return e.OriginalError
}

// IsRetryable проверяет, нужно ли повторять операцию для данной ошибки
func IsRetryable(err error, retryableErrors []error) bool {
	if err == nil {
		return false
	}

	// Если список ошибок для retry пуст, считаем все ошибки retryable
	if len(retryableErrors) == 0 {
		return true
	}

	// Проверяем, есть ли ошибка в списке retryable errors
	for _, retryableErr := range retryableErrors {
		if errors.Is(err, retryableErr) {
			return true
		}
	}

	return false
}
