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

// Statistics представляет собой потокобезопасное хранилище статистики
type Statistics struct {
	mu        sync.RWMutex
	Requests  RequestStats
	Docx      DocxStats
	Gotenberg GotenbergStats
	PDF       PDFStats
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
		TotalGenerations uint64 `json:"total_generations"`
		ErrorGenerations uint64 `json:"error_generations"`
		AverageDuration  string `json:"average_duration"`
		MinDuration      string `json:"min_duration"`
		MaxDuration      string `json:"max_duration"`
	} `json:"docx"`

	Gotenberg struct {
		TotalRequests   uint64 `json:"total_requests"`
		ErrorRequests   uint64 `json:"error_requests"`
		AverageDuration string `json:"average_duration"`
		MinDuration     string `json:"min_duration"`
		MaxDuration     string `json:"max_duration"`
	} `json:"gotenberg"`

	PDF struct {
		TotalFiles  uint64 `json:"total_files"`
		TotalSize   string `json:"total_size"`
		MinSize     string `json:"min_size"`
		MaxSize     string `json:"max_size"`
		AverageSize string `json:"average_size"`
	} `json:"pdf"`

	LastUpdated time.Time `json:"last_updated"`
}
