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

// RequestDetail представляет детальную информацию о запросе
type RequestDetail struct {
	ID               int64             `json:"id" db:"id"`
	RequestID        string            `json:"request_id" db:"request_id"`
	Timestamp        time.Time         `json:"timestamp" db:"timestamp"`
	Method           string            `json:"method" db:"method"`
	Path             string            `json:"path" db:"path"`
	ClientIP         string            `json:"client_ip" db:"client_ip"`
	UserAgent        string            `json:"user_agent" db:"user_agent"`
	Headers          map[string]string `json:"headers" db:"headers"`
	BodyText         string            `json:"body_text" db:"body_text"`
	BodySizeBytes    int64             `json:"body_size_bytes" db:"body_size_bytes"`
	Success          bool              `json:"success" db:"success"`
	HTTPStatus       int               `json:"http_status" db:"http_status"`
	DurationNs       int64             `json:"duration_ns" db:"duration_ns"`
	ContentType      string            `json:"content_type" db:"content_type"`
	HasSensitiveData bool              `json:"has_sensitive_data" db:"has_sensitive_data"`
	ErrorCategory    string            `json:"error_category" db:"error_category"`
	RequestLogID     *int64            `json:"request_log_id" db:"request_log_id"`
	DocxLogID        *int64            `json:"docx_log_id" db:"docx_log_id"`
	GotenbergLogID   *int64            `json:"gotenberg_log_id" db:"gotenberg_log_id"`
    RequestFilePath  *string           `json:"request_file_path" db:"request_file_path"`
    ResultFilePath   *string           `json:"result_file_path" db:"result_file_path"`
    ResultSizeBytes  *int64            `json:"result_size_bytes" db:"result_size_bytes"`
}

// RequestCapture представляет данные для захвата запроса
type RequestCapture struct {
	RequestID   string
	Method      string
	Path        string
	ClientIP    string
	UserAgent   string
	Headers     map[string]string
	Body        []byte
	ContentType string
	StartTime   time.Time
}

// RequestCaptureConfig настройки для захвата запросов
type RequestCaptureConfig struct {
	EnableCapture     bool     `json:"enable_capture"`
	CaptureOnlyErrors bool     `json:"capture_only_errors"`
	MaxBodySize       int64    `json:"max_body_size"`       // Максимальный размер body для сохранения
	ExcludePaths      []string `json:"exclude_paths"`       // Пути для исключения из захвата
	ExcludeHeaders    []string `json:"exclude_headers"`     // Заголовки для исключения из захвата
	RetentionDays     int      `json:"retention_days"`      // Количество дней хранения
	MaskSensitiveData bool     `json:"mask_sensitive_data"` // Маскировать чувствительные данные
    KeepLast          int      `json:"keep_last"`           // Хранить последние N запросов (артефакты)
}
