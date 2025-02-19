package statistics

import (
	"sync"
	"time"
)

// RequestStats содержит статистику по запросам
type RequestStats struct {
	TotalRequests   uint64
	SuccessRequests uint64
	FailedRequests  uint64
	TotalDuration   time.Duration
	MinDuration     time.Duration
	MaxDuration     time.Duration
	RequestsByDay   map[time.Weekday]uint64
	RequestsByHour  map[int]uint64
	LastUpdated     time.Time
}

// DocxStats содержит статистику по генерации DOCX
type DocxStats struct {
	TotalGenerations   uint64
	ErrorGenerations   uint64
	TotalDuration      time.Duration
	MinDuration        time.Duration
	MaxDuration        time.Duration
	LastGenerationTime time.Time
}

// GotenbergStats содержит статистику по работе с Gotenberg
type GotenbergStats struct {
	TotalRequests   uint64
	ErrorRequests   uint64
	TotalDuration   time.Duration
	MinDuration     time.Duration
	MaxDuration     time.Duration
	LastRequestTime time.Time
}

// PDFStats содержит статистику по PDF файлам
type PDFStats struct {
	TotalFiles        uint64
	TotalSize         int64
	MinSize           int64
	MaxSize           int64
	AverageSize       float64
	LastProcessedTime time.Time
}

// Stats представляет статистику сервиса
type Stats struct {
	Requests  RequestStats
	Docx      DocxStats
	Gotenberg GotenbergStats
	PDF       PDFStats
}

// Statistics представляет собой потокобезопасное хранилище статистики
type Statistics struct {
	mu        sync.RWMutex
	Requests  RequestStats
	Docx      DocxStats
	Gotenberg GotenbergStats
	PDF       PDFStats
	db        DB
}

// DBConfig представляет интерфейс для работы с базой данных
type DBConfig struct {
	Path string
}

// RequestLog представляет запись о запросе в БД
type RequestLog struct {
	ID        int64     `db:"id"`
	Timestamp time.Time `db:"timestamp"`
	Path      string    `db:"path"`
	Method    string    `db:"method"`
	Duration  int64     `db:"duration_ns"`
	Success   bool      `db:"success"`
}

// DocxLog представляет запись о генерации DOCX в БД
type DocxLog struct {
	ID        int64     `db:"id"`
	Timestamp time.Time `db:"timestamp"`
	Duration  int64     `db:"duration_ns"`
	HasError  bool      `db:"has_error"`
}

// GotenbergLog представляет запись о запросе к Gotenberg в БД
type GotenbergLog struct {
	ID        int64     `db:"id"`
	Timestamp time.Time `db:"timestamp"`
	Duration  int64     `db:"duration_ns"`
	HasError  bool      `db:"has_error"`
}

// PDFLog представляет запись о PDF файле в БД
type PDFLog struct {
	ID        int64     `db:"id"`
	Timestamp time.Time `db:"timestamp"`
	Size      int64     `db:"size_bytes"`
}

// StatisticsResponse представляет собой структуру ответа API
type StatisticsResponse struct {
	Requests struct {
		Total           uint64            `json:"total"`
		Success         uint64            `json:"success"`
		Failed          uint64            `json:"failed"`
		AverageDuration string            `json:"average_duration"`
		MinDuration     string            `json:"min_duration"`
		MaxDuration     string            `json:"max_duration"`
		ByDayOfWeek     map[string]uint64 `json:"by_day_of_week"`
		ByHourOfDay     map[string]uint64 `json:"by_hour_of_day"`
	} `json:"requests"`

	Docx struct {
		TotalGenerations   uint64    `json:"total_generations"`
		ErrorGenerations   uint64    `json:"error_generations"`
		AverageDuration    string    `json:"average_duration"`
		MinDuration        string    `json:"min_duration"`
		MaxDuration        string    `json:"max_duration"`
		LastGenerationTime time.Time `json:"last_generation_time"`
	} `json:"docx"`

	Gotenberg struct {
		TotalRequests   uint64    `json:"total_requests"`
		ErrorRequests   uint64    `json:"error_requests"`
		AverageDuration string    `json:"average_duration"`
		MinDuration     string    `json:"min_duration"`
		MaxDuration     string    `json:"max_duration"`
		LastRequestTime time.Time `json:"last_request_time"`
	} `json:"gotenberg"`

	PDF struct {
		TotalFiles        uint64    `json:"total_files"`
		TotalSize         string    `json:"total_size"`
		MinSize           string    `json:"min_size"`
		MaxSize           string    `json:"max_size"`
		AverageSize       string    `json:"average_size"`
		LastProcessedTime time.Time `json:"last_processed_time"`
	} `json:"pdf"`

	LastUpdated time.Time `json:"last_updated"`
}

// Config содержит конфигурацию для подключения к базе данных
type Config struct {
	Host     string
	Port     string
	DBName   string
	User     string
	Password string
}
