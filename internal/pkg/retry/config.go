package retry

import "time"

// RetryConfig содержит настройки для retry механизма
type RetryConfig struct {
	// MaxAttempts максимальное количество попыток включая первую
	MaxAttempts int
	// InitialDelay начальная задержка между попытками
	InitialDelay time.Duration
	// MaxDelay максимальная задержка между попытками
	MaxDelay time.Duration
	// BackoffFactor множитель для экспоненциальной задержки
	BackoffFactor float64
}

// ErrorType представляет тип ошибки для retry
type ErrorType string

const (
	ErrorTypeConnection ErrorType = "connection_error"
	ErrorTypeTimeout    ErrorType = "timeout"
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeUnknown    ErrorType = "unknown"
)

// RetryConfigs содержит настройки retry для разных типов ошибок
var RetryConfigs = map[ErrorType]RetryConfig{
	ErrorTypeConnection: {
		MaxAttempts:   3,
		InitialDelay:  20 * time.Millisecond,
		MaxDelay:      500 * time.Millisecond,
		BackoffFactor: 1.5,
	},
	ErrorTypeTimeout: {
		MaxAttempts:   2,
		InitialDelay:  50 * time.Millisecond,
		MaxDelay:      200 * time.Millisecond,
		BackoffFactor: 2,
	},
	ErrorTypeValidation: {
		MaxAttempts:   1, // Не ретраим валидационные ошибки
		InitialDelay:  0,
		MaxDelay:      0,
		BackoffFactor: 1,
	},
	ErrorTypeUnknown: {
		MaxAttempts:   2,
		InitialDelay:  20 * time.Millisecond,
		MaxDelay:      300 * time.Millisecond,
		BackoffFactor: 1.5,
	},
}

// GetRetryConfig возвращает конфигурацию retry для заданного типа ошибки
func GetRetryConfig(errorType ErrorType) RetryConfig {
	if config, ok := RetryConfigs[errorType]; ok {
		return config
	}
	return RetryConfigs[ErrorTypeUnknown]
}

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
		MaxAttempts:   2,
		InitialDelay:  50 * time.Millisecond,
		MaxDelay:      200 * time.Millisecond,
		BackoffFactor: 1.5,
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
