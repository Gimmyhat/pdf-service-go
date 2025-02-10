package retry

import (
	"errors"
	"net"
	"os"
	"syscall"
)

// TimeoutError интерфейс для ошибок таймаута
type TimeoutError interface {
	Timeout() bool
}

// ConnectionError интерфейс для ошибок соединения
type ConnectionError interface {
	Connection() bool
}

// ValidationError интерфейс для ошибок валидации
type ValidationError interface {
	Validation() bool
}

// IsTimeout проверяет, является ли ошибка таймаутом
func IsTimeout(err error) bool {
	var timeoutErr TimeoutError
	if errors.As(err, &timeoutErr) {
		return timeoutErr.Timeout()
	}

	// Проверяем стандартные ошибки таймаута
	if os.IsTimeout(err) {
		return true
	}

	// Проверяем сетевые таймауты
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	// Проверяем системные таймауты
	var sysErr syscall.Errno
	if errors.As(err, &sysErr) {
		return sysErr == syscall.ETIMEDOUT
	}

	return false
}

// IsConnectionError проверяет, является ли ошибка проблемой соединения
func IsConnectionError(err error) bool {
	var connErr ConnectionError
	if errors.As(err, &connErr) {
		return connErr.Connection()
	}

	// Проверяем сетевые ошибки
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Проверяем системные ошибки соединения
	var sysErr syscall.Errno
	if errors.As(err, &sysErr) {
		switch sysErr {
		case syscall.ECONNREFUSED,
			syscall.ECONNRESET,
			syscall.ECONNABORTED,
			syscall.ENETUNREACH,
			syscall.ENETDOWN:
			return true
		}
	}

	return false
}

// IsValidationError проверяет, является ли ошибка проблемой валидации
func IsValidationError(err error) bool {
	var validErr ValidationError
	if errors.As(err, &validErr) {
		return validErr.Validation()
	}

	// Здесь можно добавить дополнительные проверки для конкретных типов ошибок валидации
	return false
}

// IsTransientError проверяет, является ли ошибка временной
func IsTransientError(err error) bool {
	return IsTimeout(err) || IsConnectionError(err)
}

// ShouldRetry определяет, нужно ли повторять операцию для данной ошибки
func ShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Не повторяем для ошибок валидации
	if IsValidationError(err) {
		return false
	}

	// Повторяем для временных ошибок
	if IsTransientError(err) {
		return true
	}

	return false
}
