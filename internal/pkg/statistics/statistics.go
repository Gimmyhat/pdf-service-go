package statistics

import (
	"fmt"
	"sync"
	"time"
)

var (
	instance *Statistics
	once     sync.Once
)

// New создает новый экземпляр DB
func New(cfg Config) (DB, error) {
	return NewPostgresDB(cfg.Host, cfg.Port, cfg.DBName, cfg.User, cfg.Password)
}

// NewStatistics создает новый экземпляр Statistics
func NewStatistics(db DB) *Statistics {
	return &Statistics{db: db}
}

// GetInstance возвращает синглтон Statistics
func GetInstance() *Statistics {
	return instance
}

// Initialize инициализирует синглтон Statistics
func Initialize(cfg Config) error {
	var err error
	once.Do(func() {
		var db DB
		db, err = New(cfg)
		if err != nil {
			return
		}
		instance = NewStatistics(db)
	})
	return err
}

// TrackRequest записывает информацию о запросе
func (s *Statistics) TrackRequest(path, method string, duration time.Duration, success bool) error {
	return s.db.LogRequest(time.Now(), path, method, duration, success)
}

// TrackDocx записывает информацию о генерации DOCX
func (s *Statistics) TrackDocx(duration time.Duration, hasError bool) error {
	return s.db.LogDocx(time.Now(), duration, hasError)
}

// TrackGotenberg записывает информацию о запросе к Gotenberg
func (s *Statistics) TrackGotenberg(duration time.Duration, hasError bool) error {
	return s.db.LogGotenberg(time.Now(), duration, hasError)
}

// TrackPDF записывает информацию о PDF файле
func (s *Statistics) TrackPDF(size int64) error {
	return s.db.LogPDF(time.Now(), size)
}

// GetStatistics возвращает статистику за указанный период
func (s *Statistics) GetStatistics(since time.Time) (*Stats, error) {
	return s.db.GetStatistics(since)
}

// Close закрывает соединение с базой данных
func (s *Statistics) Close() error {
	return s.db.Close()
}

// GetStatisticsForPeriod возвращает статистику за указанный период в формате для API
func (s *Statistics) GetStatisticsForPeriod(period string) (StatisticsResponse, error) {
	var since time.Time
	now := time.Now()

	// Определяем период
	switch period {
	case "15min":
		since = now.Add(-15 * time.Minute)
	case "1hour":
		since = now.Add(-1 * time.Hour)
	case "5hours":
		since = now.Add(-5 * time.Hour)
	case "day":
		since = now.Add(-24 * time.Hour)
	case "week":
		since = now.Add(-7 * 24 * time.Hour)
	case "month":
		since = now.Add(-30 * 24 * time.Hour)
	default:
		// Для "all" или неизвестного периода берем всю статистику
		since = time.Time{} // Нулевое время = начало времен
	}

	stats, err := s.db.GetStatistics(since)
	if err != nil {
		return StatisticsResponse{}, fmt.Errorf("failed to get statistics: %w", err)
	}

	var response StatisticsResponse

	// Заполняем статистику запросов
	response.Requests.Total = stats.Requests.TotalRequests
	response.Requests.Success = stats.Requests.SuccessRequests
	response.Requests.Failed = stats.Requests.FailedRequests

	if stats.Requests.TotalRequests > 0 {
		avgDuration := stats.Requests.TotalDuration / time.Duration(stats.Requests.TotalRequests)
		response.Requests.AverageDuration = avgDuration.String()
		response.Requests.MinDuration = stats.Requests.MinDuration.String()
		response.Requests.MaxDuration = stats.Requests.MaxDuration.String()
	}

	// Заполняем статистику DOCX
	response.Docx.TotalGenerations = stats.Docx.TotalGenerations
	response.Docx.ErrorGenerations = stats.Docx.ErrorGenerations
	if stats.Docx.TotalGenerations > 0 {
		avgDuration := stats.Docx.TotalDuration / time.Duration(stats.Docx.TotalGenerations)
		response.Docx.AverageDuration = avgDuration.String()
		response.Docx.MinDuration = stats.Docx.MinDuration.String()
		response.Docx.MaxDuration = stats.Docx.MaxDuration.String()
	}
	response.Docx.LastGenerationTime = stats.Docx.LastGenerationTime

	// Заполняем статистику Gotenberg
	response.Gotenberg.TotalRequests = stats.Gotenberg.TotalRequests
	response.Gotenberg.ErrorRequests = stats.Gotenberg.ErrorRequests
	if stats.Gotenberg.TotalRequests > 0 {
		avgDuration := stats.Gotenberg.TotalDuration / time.Duration(stats.Gotenberg.TotalRequests)
		response.Gotenberg.AverageDuration = avgDuration.String()
		response.Gotenberg.MinDuration = stats.Gotenberg.MinDuration.String()
		response.Gotenberg.MaxDuration = stats.Gotenberg.MaxDuration.String()
	}
	response.Gotenberg.LastRequestTime = stats.Gotenberg.LastRequestTime

	// Заполняем статистику PDF
	response.PDF.TotalFiles = stats.PDF.TotalFiles
	if stats.PDF.TotalFiles > 0 {
		response.PDF.TotalSize = formatBytes(stats.PDF.TotalSize)
		response.PDF.MinSize = formatBytes(stats.PDF.MinSize)
		response.PDF.MaxSize = formatBytes(stats.PDF.MaxSize)
		response.PDF.AverageSize = formatBytes(int64(stats.PDF.AverageSize))
	}
	response.PDF.LastProcessedTime = stats.PDF.LastProcessedTime

	// Конвертируем дни недели в строки
	response.Requests.ByDayOfWeek = make(map[string]uint64)
	weekdayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	for day, count := range stats.Requests.RequestsByDay {
		response.Requests.ByDayOfWeek[weekdayNames[day]] = count
	}

	// Конвертируем часы в строки
	response.Requests.ByHourOfDay = make(map[string]uint64)
	for hour, count := range stats.Requests.RequestsByHour {
		response.Requests.ByHourOfDay[fmt.Sprintf("%02d:00", hour)] = count
	}

	response.LastUpdated = stats.Requests.LastUpdated

	return response, nil
}

// formatBytes форматирует размер в байтах в человекочитаемый формат
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
