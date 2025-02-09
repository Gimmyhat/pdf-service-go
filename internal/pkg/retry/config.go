package retry

import "time"

// Config содержит настройки для механизма retry
type Config struct {
	// MaxAttempts максимальное количество попыток включая первую
	MaxAttempts int
	// InitialDelay начальная задержка между попытками
	InitialDelay time.Duration
	// MaxDelay максимальная задержка между попытками
	MaxDelay time.Duration
	// BackoffFactor множитель для экспоненциальной задержки
	BackoffFactor float64
	// RetryableErrors список ошибок, для которых нужно выполнять retry
	RetryableErrors []error
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	return &Config{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      2 * time.Second,
		BackoffFactor: 2.0,
	}
}

// Option функциональный опции для конфигурации
type Option func(*Config)

// WithMaxAttempts устанавливает максимальное количество попыток
func WithMaxAttempts(attempts int) Option {
	return func(c *Config) {
		c.MaxAttempts = attempts
	}
}

// WithInitialDelay устанавливает начальную задержку
func WithInitialDelay(delay time.Duration) Option {
	return func(c *Config) {
		c.InitialDelay = delay
	}
}

// WithMaxDelay устанавливает максимальную задержку
func WithMaxDelay(delay time.Duration) Option {
	return func(c *Config) {
		c.MaxDelay = delay
	}
}

// WithBackoffFactor устанавливает множитель для экспоненциальной задержки
func WithBackoffFactor(factor float64) Option {
	return func(c *Config) {
		c.BackoffFactor = factor
	}
}

// WithRetryableErrors устанавливает список ошибок для retry
func WithRetryableErrors(errors []error) Option {
	return func(c *Config) {
		c.RetryableErrors = errors
	}
}
