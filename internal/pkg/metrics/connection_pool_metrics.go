package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ConnectionPoolTotalConnections общее количество соединений в пуле
	ConnectionPoolTotalConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "connection_pool_total_connections",
			Help: "Total number of connections in the pool",
		},
	)

	// ConnectionPoolActiveConnections количество активных соединений
	ConnectionPoolActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "connection_pool_active_connections",
			Help: "Number of active connections in the pool",
		},
	)

	// ConnectionPoolGetDuration время получения соединения из пула
	ConnectionPoolGetDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "connection_pool_get_duration_seconds",
			Help:    "Time taken to get a connection from the pool",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
		},
	)

	// ConnectionPoolErrors ошибки пула соединений
	ConnectionPoolErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "connection_pool_errors_total",
			Help: "Total number of connection pool errors",
		},
		[]string{"type"},
	)

	// ConnectionPoolCreatedConnections количество созданных соединений
	ConnectionPoolCreatedConnections = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "connection_pool_created_connections_total",
			Help: "Total number of connections created",
		},
	)

	// ConnectionPoolRemovedConnections количество удаленных соединений
	ConnectionPoolRemovedConnections = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "connection_pool_removed_connections_total",
			Help: "Total number of connections removed",
		},
		[]string{"reason"},
	)

	// ConnectionPoolWaitingRequests количество запросов, ожидающих соединение
	ConnectionPoolWaitingRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "connection_pool_waiting_requests",
			Help: "Number of requests waiting for a connection",
		},
	)

	// ConnectionPoolIdleTime время простоя соединений
	ConnectionPoolIdleTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "connection_pool_idle_time_seconds",
			Help:    "Time connections spend idle in the pool",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10),
		},
		[]string{"state"},
	)
)
