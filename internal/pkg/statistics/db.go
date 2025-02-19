package statistics

import (
	"time"
)

// DB представляет интерфейс для работы с базой данных статистики
type DB interface {
	// LogRequest записывает информацию о запросе
	LogRequest(timestamp time.Time, path, method string, duration time.Duration, success bool) error

	// LogDocx записывает информацию о генерации DOCX
	LogDocx(timestamp time.Time, duration time.Duration, hasError bool) error

	// LogGotenberg записывает информацию о запросе к Gotenberg
	LogGotenberg(timestamp time.Time, duration time.Duration, hasError bool) error

	// LogPDF записывает информацию о PDF файле
	LogPDF(timestamp time.Time, size int64) error

	// GetStatistics возвращает статистику за указанный период
	GetStatistics(since time.Time) (*Stats, error)

	// Close закрывает соединение с базой данных
	Close() error
}
