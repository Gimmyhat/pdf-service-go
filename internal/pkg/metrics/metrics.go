package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestsTotal точное количество запросов на генерацию PDF
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pdf_requests_total",
			Help: "Total number of PDF generation requests",
		},
		[]string{"status"},
	)

	// HTTPRequestsTotal количество HTTP запросов
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration длительность HTTP запросов
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// PDFGenerationDuration длительность генерации PDF
	PDFGenerationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pdf_generation_duration_seconds",
			Help:    "Duration of PDF generation in seconds",
			Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10, 20, 30, 60},
		},
		[]string{"template"},
	)

	// PDFFileSizeBytes размер сгенерированных PDF файлов
	PDFFileSizeBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pdf_file_size_bytes",
			Help:    "Size of generated PDF files in bytes",
			Buckets: []float64{1024, 10 * 1024, 100 * 1024, 1024 * 1024, 10 * 1024 * 1024},
		},
		[]string{"operation"},
	)

	// GotenbergRequestsTotal количество запросов к Gotenberg
	GotenbergRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gotenberg_requests_total",
			Help: "Total number of requests to Gotenberg service",
		},
		[]string{"status"},
	)

	// GotenbergRequestDuration длительность запросов к Gotenberg
	GotenbergRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gotenberg_request_duration_seconds",
			Help:    "Duration of Gotenberg requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	// RetryAttemptsTotal общее количество попыток retry по операциям
	RetryAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "retry_attempts_total",
			Help: "Total number of retry attempts by operation",
		},
		[]string{"operation", "attempt", "status"},
	)

	// RetryOperationDuration длительность операций с учетом retry
	RetryOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "retry_operation_duration_seconds",
			Help:    "Duration of retry operations by attempt",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 10),
		},
		[]string{"operation", "attempt", "status"},
	)

	// RetryErrorsTotal количество ошибок по типам
	RetryErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "retry_errors_total",
			Help: "Total number of retry errors by type",
		},
		[]string{"operation", "error_type", "attempt"},
	)

	// RetryBackoffDuration длительность задержек между попытками
	RetryBackoffDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "retry_backoff_duration_seconds",
			Help:    "Duration of retry backoff by attempt",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 8),
		},
		[]string{"operation", "attempt"},
	)

	// RetrySuccessRate процент успешных операций после retry
	RetrySuccessRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "retry_success_rate",
			Help: "Success rate of operations after retries",
		},
		[]string{"operation"},
	)

	// RetryCurrentAttempts текущее количество операций в retry
	RetryCurrentAttempts = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "retry_current_attempts",
			Help: "Current number of operations in retry state",
		},
		[]string{"operation"},
	)
)
