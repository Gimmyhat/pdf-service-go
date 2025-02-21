package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RetryReasonTotal количество retry по причинам
	RetryReasonTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "retry_reason_total",
			Help: "Total number of retries by reason",
		},
		[]string{"operation", "reason"},
	)

	// RetryTotalDuration общая длительность операции с учетом всех retry
	RetryTotalDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "retry_total_duration_seconds",
			Help:    "Total duration of operation including all retries",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
		},
		[]string{"operation", "success"},
	)

	// RetryAttemptsDistribution распределение количества попыток
	RetryAttemptsDistribution = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "retry_attempts_distribution",
			Help:    "Distribution of retry attempts count",
			Buckets: []float64{1, 2, 3, 4, 5},
		},
		[]string{"operation"},
	)

	// RetryConsecutiveFailures количество последовательных неудачных попыток
	RetryConsecutiveFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "retry_consecutive_failures",
			Help: "Number of consecutive retry failures",
		},
		[]string{"operation"},
	)

	// RetryRecoveryTime время восстановления после серии неудачных попыток
	RetryRecoveryTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "retry_recovery_time_seconds",
			Help:    "Time taken to recover after consecutive failures",
			Buckets: prometheus.ExponentialBuckets(1, 2, 8),
		},
		[]string{"operation"},
	)
)
