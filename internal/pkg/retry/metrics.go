package retry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	retryAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "retry_attempts_total",
			Help: "Total number of retry attempts",
		},
		[]string{"operation"},
	)

	retrySuccess = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "retry_success_total",
			Help: "Total number of successful retries",
		},
		[]string{"operation"},
	)

	retryFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "retry_failures_total",
			Help: "Total number of failed retries",
		},
		[]string{"operation"},
	)

	retryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "retry_duration_seconds",
			Help:    "Time spent retrying operations",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 10),
		},
		[]string{"operation"},
	)
)

// метрики для конкретной операции
type metrics struct {
	operation string
	attempts  prometheus.Counter
	success   prometheus.Counter
	failures  prometheus.Counter
	duration  prometheus.Observer
}

func newMetrics(operation string) *metrics {
	return &metrics{
		operation: operation,
		attempts:  retryAttempts.WithLabelValues(operation),
		success:   retrySuccess.WithLabelValues(operation),
		failures:  retryFailures.WithLabelValues(operation),
		duration:  retryDuration.WithLabelValues(operation),
	}
}

func (m *metrics) recordAttempt() {
	m.attempts.Inc()
}

func (m *metrics) recordSuccess() {
	m.success.Inc()
}

func (m *metrics) recordFailure() {
	m.failures.Inc()
}

func (m *metrics) recordDuration(duration float64) {
	m.duration.Observe(duration)
}
