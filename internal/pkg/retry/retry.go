package retry

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Operation представляет операцию, которую нужно повторить
type Operation func(ctx context.Context) error

// Retrier выполняет повторные попытки операции
type Retrier struct {
	config  *Config
	logger  *zap.Logger
	metrics *metrics
}

// New создает новый экземпляр Retrier
func New(operation string, logger *zap.Logger, opts ...Option) *Retrier {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	return &Retrier{
		config:  config,
		logger:  logger,
		metrics: newMetrics(operation),
	}
}

// Do выполняет операцию с повторными попытками
func (r *Retrier) Do(ctx context.Context, op Operation) error {
	start := time.Now()
	var lastErr error

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		r.metrics.recordAttempt()

		err := op(ctx)
		if err == nil {
			r.metrics.recordSuccess()
			r.metrics.recordDuration(time.Since(start).Seconds())
			return nil
		}

		lastErr = err
		r.logger.Warn("retry attempt failed",
			zap.Int("attempt", attempt),
			zap.Error(err),
		)

		// Если контекст отменен, прекращаем попытки
		if ctx.Err() != nil {
			r.metrics.recordFailure()
			r.metrics.recordDuration(time.Since(start).Seconds())
			return ctx.Err()
		}

		// Если ошибка не подлежит retry, прекращаем попытки
		if !IsRetryable(err, r.config.RetryableErrors) {
			r.metrics.recordFailure()
			r.metrics.recordDuration(time.Since(start).Seconds())
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

		// Ждем с учетом контекста
		select {
		case <-ctx.Done():
			r.metrics.recordFailure()
			r.metrics.recordDuration(time.Since(start).Seconds())
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	r.metrics.recordFailure()
	r.metrics.recordDuration(time.Since(start).Seconds())

	if lastErr != nil {
		return &RetryError{
			Attempt:       r.config.MaxAttempts,
			OriginalError: lastErr,
		}
	}

	return ErrMaxAttemptsReached
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
