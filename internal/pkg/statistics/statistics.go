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

// GetPostgresDB возвращает PostgresDB из statistics instance
func GetPostgresDB() *PostgresDB {
	if instance != nil && instance.db != nil {
		if pgDB, ok := instance.db.(*PostgresDB); ok {
			return pgDB
		}
	}
	return nil
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

// InitializeOrRetry пытается инициализировать Statistics, если она ещё не инициализирована.
// В отличие от Initialize, может вызываться многократно до успешного подключения.
var initMu sync.Mutex

func InitializeOrRetry(cfg Config) error {
    initMu.Lock()
    defer initMu.Unlock()
    if instance != nil {
        return nil
    }
    db, err := New(cfg)
    if err != nil {
        return err
    }
    instance = NewStatistics(db)
    return nil
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

// GetDB возвращает интерфейс базы данных для прямого доступа
func (s *Statistics) GetDB() DB {
	return s.db
}

// LogError записывает детальную информацию об ошибке
func (s *Statistics) LogError(errorDetails *ErrorDetails) error {
	return s.db.LogError(errorDetails)
}

// GetErrorSummary возвращает сводку ошибок
func (s *Statistics) GetErrorSummary(since time.Time, limit int) (*ErrorSummary, error) {
	// Получаем последние ошибки
	recentErrors, err := s.db.GetRecentErrors(limit, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent errors: %w", err)
	}

	// Получаем паттерны ошибок
	patterns, err := s.db.GetErrorPatterns(since)
	if err != nil {
		return nil, fmt.Errorf("failed to get error patterns: %w", err)
	}

	// Получаем счетчики
	total, last24h, lastHour, err := s.db.GetErrorCounts()
	if err != nil {
		return nil, fmt.Errorf("failed to get error counts: %w", err)
	}

	// Добавляем решения к ошибкам
	for i := range recentErrors {
		recentErrors[i].RequestDetails["solutions"] = GetErrorSolutions(
			recentErrors[i].ErrorType,
			recentErrors[i].Component,
		)
	}

	return &ErrorSummary{
		RecentErrors:   recentErrors,
		ErrorPatterns:  patterns,
		TotalErrors:    total,
		ErrorsLast24h:  last24h,
		ErrorsLastHour: lastHour,
		TopErrorTypes:  patterns, // Уже отсортированы по количеству
	}, nil
}

// Типы ответов API
type RequestsResponse struct {
	Total           uint64
	Success         uint64
	Failed          uint64
	AverageDuration string
	ByDayOfWeek     map[string]uint64
	ByHourOfDay     map[string]uint64
}

type DocxResponse struct {
	TotalGenerations   uint64
	ErrorGenerations   uint64
	AverageDuration    string
	LastGenerationTime string
}

type GotenbergResponse struct {
	TotalRequests   uint64
	ErrorRequests   uint64
	AverageDuration string
	LastRequestTime string
}

type PDFResponse struct {
	TotalFiles        uint64
	AverageSize       string
	MinSize           string
	MaxSize           string
	LastProcessedTime string
}

// GetAverageDuration возвращает среднюю продолжительность для RequestStats
func (s *RequestStats) GetAverageDuration() time.Duration {
	if s.TotalRequests == 0 {
		return 0
	}
	return s.TotalDuration / time.Duration(s.TotalRequests)
}

// GetAverageDuration возвращает среднюю продолжительность для DocxStats
func (s *DocxStats) GetAverageDuration() time.Duration {
	if s.TotalGenerations == 0 {
		return 0
	}
	return s.TotalDuration / time.Duration(s.TotalGenerations)
}

// GetAverageDuration возвращает среднюю продолжительность для GotenbergStats
func (s *GotenbergStats) GetAverageDuration() time.Duration {
	if s.TotalRequests == 0 {
		return 0
	}
	return s.TotalDuration / time.Duration(s.TotalRequests)
}

// GetAverageSize возвращает средний размер для PDFStats
func (s *PDFStats) GetAverageSize() int64 {
	if s.TotalFiles == 0 {
		return 0
	}
	return s.TotalSize / int64(s.TotalFiles)
}

// GetStatisticsForPeriod возвращает статистику за указанный период
func (s *Statistics) GetStatisticsForPeriod(period string) (*StatisticsResponse, error) {
	// Загружаем московскую временную зону
	moscowLoc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return nil, fmt.Errorf("failed to load Moscow timezone: %w", err)
	}

	// Получаем текущее время в московской зоне
	now := time.Now().In(moscowLoc)
	var since time.Time

	// Определяем начальное время периода
	switch period {
	case "15min":
		since = now.Add(-15 * time.Minute)
	case "1hour":
		since = now.Add(-1 * time.Hour)
	case "5hours":
		since = now.Add(-5 * time.Hour)
	case "24hours":
		since = now.Add(-24 * time.Hour)
	case "week":
		since = now.AddDate(0, 0, -7)
	case "month":
		since = now.AddDate(0, -1, 0)
	case "all", "":
		// Для "all" или пустого значения используем нулевое время
		since = time.Time{}
	default:
		return nil, fmt.Errorf("unknown period: %s", period)
	}

	// Получаем статистику из базы данных
	stats, err := s.db.GetStatistics(since)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	// Преобразуем статистику в ответ API
	response := &StatisticsResponse{}

	// Заполняем статистику запросов
	response.Requests.Total = stats.Requests.TotalRequests
	response.Requests.Success = stats.Requests.SuccessRequests
	response.Requests.Failed = stats.Requests.FailedRequests
	response.Requests.AverageDuration = formatDuration(stats.Requests.GetAverageDuration())
	response.Requests.MinDuration = formatDuration(stats.Requests.MinDuration)
	response.Requests.MaxDuration = formatDuration(stats.Requests.MaxDuration)
	response.Requests.ByDayOfWeek = make(map[string]uint64)
	response.Requests.ByHourOfDay = make(map[string]uint64)

	// Заполняем статистику DOCX
	response.Docx.TotalGenerations = stats.Docx.TotalGenerations
	response.Docx.ErrorGenerations = stats.Docx.ErrorGenerations
	response.Docx.AverageDuration = formatDuration(stats.Docx.GetAverageDuration())
	response.Docx.MinDuration = formatDuration(stats.Docx.MinDuration)
	response.Docx.MaxDuration = formatDuration(stats.Docx.MaxDuration)
	response.Docx.LastGenerationTime = stats.Docx.LastGenerationTime

	// Заполняем статистику Gotenberg
	response.Gotenberg.TotalRequests = stats.Gotenberg.TotalRequests
	response.Gotenberg.ErrorRequests = stats.Gotenberg.ErrorRequests
	response.Gotenberg.AverageDuration = formatDuration(stats.Gotenberg.GetAverageDuration())
	response.Gotenberg.MinDuration = formatDuration(stats.Gotenberg.MinDuration)
	response.Gotenberg.MaxDuration = formatDuration(stats.Gotenberg.MaxDuration)
	response.Gotenberg.LastRequestTime = stats.Gotenberg.LastRequestTime

	// Заполняем статистику PDF
	response.PDF.TotalFiles = stats.PDF.TotalFiles
	response.PDF.TotalSize = formatBytes(stats.PDF.TotalSize)
	response.PDF.AverageSize = formatBytes(stats.PDF.GetAverageSize())
	response.PDF.MinSize = formatBytes(stats.PDF.MinSize)
	response.PDF.MaxSize = formatBytes(stats.PDF.MaxSize)
	response.PDF.LastProcessedTime = stats.PDF.LastProcessedTime

	// Преобразуем статистику по дням недели
	for day, count := range stats.Requests.RequestsByDay {
		response.Requests.ByDayOfWeek[day.String()] = count
	}

	// Преобразуем статистику по часам
	for hour, count := range stats.Requests.RequestsByHour {
		response.Requests.ByHourOfDay[fmt.Sprintf("%02d", hour)] = count
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

// formatDuration форматирует продолжительность в человекочитаемый формат
func formatDuration(duration time.Duration) string {
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// formatTime форматирует время в строку в формате ISO 8601
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
