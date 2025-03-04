package retry

import (
	"context"
	"strconv"
	"time"

	"pdf-service-go/internal/pkg/metrics"

	"go.uber.org/zap"
)

// Operation представляет операцию, которую нужно повторить
type Operation func(ctx context.Context) error

// Retrier выполняет повторные попытки операции
type Retrier struct {
	config    *Config
	logger    *zap.Logger
	operation string
}

// New создает новый экземпляр Retrier
func New(operation string, logger *zap.Logger, opts ...Option) *Retrier {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	return &Retrier{
		config:    config,
		logger:    logger,
		operation: operation,
	}
}

// Do выполняет операцию с повторными попытками
func (r *Retrier) Do(ctx context.Context, op Operation) error {
	start := time.Now()
	var lastErr error
	consecutiveFailures := 0
	lastFailureTime := time.Now()

	// Увеличиваем счетчик текущих операций в retry
	metrics.RetryCurrentAttempts.WithLabelValues(r.operation).Inc()
	defer metrics.RetryCurrentAttempts.WithLabelValues(r.operation).Dec()

	successfulAttempts := 0
	totalAttempts := 0

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		attemptStr := strconv.Itoa(attempt)
		attemptStart := time.Now()
		totalAttempts++

		// Записываем попытку в метрики
		metrics.RetryAttemptsTotal.WithLabelValues(r.operation, attemptStr, "started").Inc()

		err := op(ctx)
		attemptDuration := time.Since(attemptStart)

		// Записываем длительность операции
		metrics.RetryOperationDuration.WithLabelValues(
			r.operation,
			attemptStr,
			errorToStatus(err, ctx),
		).Observe(attemptDuration.Seconds())

		if err == nil {
			// Успешное выполнение
			successfulAttempts++
			metrics.RetryAttemptsTotal.WithLabelValues(r.operation, attemptStr, "success").Inc()
			metrics.RetryOperationDuration.WithLabelValues(r.operation, attemptStr, "success").Observe(time.Since(start).Seconds())

			// Если были предыдущие ошибки, записываем время восстановления
			if consecutiveFailures > 0 {
				recoveryTime := time.Since(lastFailureTime)
				metrics.RetryRecoveryTime.WithLabelValues(r.operation).Observe(recoveryTime.Seconds())
			}

			// Обновляем процент успешных операций
			successRate := float64(successfulAttempts) / float64(totalAttempts)
			metrics.RetrySuccessRate.WithLabelValues(r.operation).Set(successRate)

			// Записываем общую длительность успешной операции
			metrics.RetryTotalDuration.WithLabelValues(r.operation, "true").Observe(time.Since(start).Seconds())

			// Записываем распределение попыток
			metrics.RetryAttemptsDistribution.WithLabelValues(r.operation).Observe(float64(attempt))

			return nil
		}

		lastErr = err
		consecutiveFailures++
		lastFailureTime = time.Now()

		// Увеличиваем счетчик последовательных ошибок
		metrics.RetryConsecutiveFailures.WithLabelValues(r.operation).Inc()

		r.logger.Warn("retry attempt failed",
			zap.Int("attempt", attempt),
			zap.Error(err),
			zap.Duration("duration", attemptDuration),
		)

		// Записываем ошибку в метрики
		metrics.RetryAttemptsTotal.WithLabelValues(r.operation, attemptStr, "failed").Inc()
		errorType := classifyError(err, ctx)
		metrics.RetryErrorsTotal.WithLabelValues(
			r.operation,
			errorType,
			attemptStr,
		).Inc()

		// Записываем причину retry
		retryReason := determineRetryReason(err, ctx)
		metrics.RetryReasonTotal.WithLabelValues(r.operation, retryReason).Inc()

		// Если контекст отменен, прекращаем попытки
		if ctx.Err() != nil {
			metrics.RetryAttemptsTotal.WithLabelValues(r.operation, attemptStr, "cancelled").Inc()
			metrics.RetryOperationDuration.WithLabelValues(r.operation, attemptStr, "cancelled").Observe(time.Since(start).Seconds())
			metrics.RetryTotalDuration.WithLabelValues(r.operation, "false").Observe(time.Since(start).Seconds())
			return ctx.Err()
		}

		// Если ошибка не подлежит retry, прекращаем попытки
		if !IsRetryable(err, r.config.RetryableErrors) {
			metrics.RetryAttemptsTotal.WithLabelValues(r.operation, attemptStr, "non_retryable").Inc()
			metrics.RetryOperationDuration.WithLabelValues(r.operation, attemptStr, "non_retryable").Observe(time.Since(start).Seconds())
			metrics.RetryTotalDuration.WithLabelValues(r.operation, "false").Observe(time.Since(start).Seconds())
			return &RetryError{
				Attempt:       attempt,
				OriginalError: err,
			}
		}

		// Если это последняя попытка, не нужно ждать
		if attempt == r.config.MaxAttempts {
			break
		}

		// Вычисляем задержку для следующей попытки
		delay := r.calculateDelay(attempt)

		// Записываем длительность задержки
		metrics.RetryBackoffDuration.WithLabelValues(
			r.operation,
			attemptStr,
		).Observe(delay.Seconds())

		// Ждем с учетом контекста
		select {
		case <-ctx.Done():
			metrics.RetryAttemptsTotal.WithLabelValues(r.operation, attemptStr, "cancelled").Inc()
			metrics.RetryOperationDuration.WithLabelValues(r.operation, attemptStr, "cancelled").Observe(time.Since(start).Seconds())
			metrics.RetryTotalDuration.WithLabelValues(r.operation, "false").Observe(time.Since(start).Seconds())
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	// Обновляем процент успешных операций
	successRate := float64(successfulAttempts) / float64(totalAttempts)
	metrics.RetrySuccessRate.WithLabelValues(r.operation).Set(successRate)

	// Записываем общую длительность неуспешной операции
	metrics.RetryTotalDuration.WithLabelValues(r.operation, "false").Observe(time.Since(start).Seconds())

	// Записываем распределение попыток
	metrics.RetryAttemptsDistribution.WithLabelValues(r.operation).Observe(float64(r.config.MaxAttempts))

	if lastErr != nil {
		metrics.RetryAttemptsTotal.WithLabelValues(r.operation, strconv.Itoa(r.config.MaxAttempts), "max_attempts").Inc()
		return &RetryError{
			Attempt:       r.config.MaxAttempts,
			OriginalError: lastErr,
		}
	}

	return ErrMaxAttemptsReached
}

// determineRetryReason определяет причину retry
func determineRetryReason(err error, ctx context.Context) string {
	if ctx.Err() != nil {
		return "context_cancelled"
	}
	if IsTimeout(err) {
		return "timeout"
	}
	if IsConnectionError(err) {
		return "connection_error"
	}
	if IsValidationError(err) {
		return "validation_error"
	}
	return "unknown"
}

// calculateDelay вычисляет задержку для следующей попытки
func (r *Retrier) calculateDelay(attempt int) time.Duration {
	delay := float64(r.config.InitialDelay)
	for i := 1; i < attempt; i++ {
		delay *= r.config.BackoffFactor
	}

	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	return time.Duration(delay)
}

// errorToStatus преобразует ошибку в статус для метрик
func errorToStatus(err error, ctx context.Context) string {
	if err == nil {
		return "success"
	}
	if ctx.Err() != nil {
		return "cancelled"
	}
	return "error"
}

// classifyError классифицирует ошибку для метрик
func classifyError(err error, ctx context.Context) string {
	if err == nil {
		return "none"
	}

	switch {
	case ctx.Err() != nil:
		return "context_cancelled"
	case IsTimeout(err):
		return "timeout"
	case IsConnectionError(err):
		return "connection"
	case IsValidationError(err):
		return "validation"
	default:
		return "unknown"
	}
}

// UpdateConfig обновляет конфигурацию retry
func (r *Retrier) UpdateConfig(config RetryConfig) {
	r.config.MaxAttempts = config.MaxAttempts
	r.config.InitialDelay = config.InitialDelay
	r.config.MaxDelay = config.MaxDelay
	r.config.BackoffFactor = config.BackoffFactor
}
