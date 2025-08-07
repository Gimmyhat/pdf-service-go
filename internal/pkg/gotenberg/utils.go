package gotenberg

import (
	"os"
	"strconv"
	"time"
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