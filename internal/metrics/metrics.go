package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestsTotal считает общее количество запросов
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pdf_service_requests_total",
			Help: "The total number of processed requests",
		},
		[]string{"status", "operation"},
	)

	// RequestDuration измеряет длительность запросов
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pdf_service_request_duration_seconds",
			Help:    "The duration of requests in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
		},
		[]string{"operation"},
	)

	// FileSize измеряет размеры генерируемых файлов
	FileSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pdf_service_file_size_bytes",
			Help:    "The size of generated files in bytes",
			Buckets: []float64{1e5, 5e5, 1e6, 5e6, 1e7, 5e7}, // 100KB, 500KB, 1MB, 5MB, 10MB, 50MB
		},
		[]string{"type"},
	)

	// ErrorsTotal считает количество ошибок
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pdf_service_errors_total",
			Help: "The total number of errors",
		},
		[]string{"type", "operation"},
	)

	// GotenbergRequestsTotal считает запросы к Gotenberg
	GotenbergRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pdf_service_gotenberg_requests_total",
			Help: "The total number of requests to Gotenberg",
		},
		[]string{"status"},
	)

	// GotenbergRequestDuration измеряет длительность запросов к Gotenberg
	GotenbergRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pdf_service_gotenberg_request_duration_seconds",
			Help:    "The duration of Gotenberg requests in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
		},
		[]string{"operation"},
	)
)
