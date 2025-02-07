package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
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

	// PDFGenerationTotal количество сгенерированных PDF
	PDFGenerationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pdf_generation_total",
			Help: "Total number of PDF generations",
		},
		[]string{"status"},
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
		[]string{"template"},
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
)
