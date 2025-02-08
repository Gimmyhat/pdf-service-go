package cache

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	cacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "template_cache_hits_total",
			Help: "Total number of template cache hits",
		},
		[]string{"template"},
	)

	cacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "template_cache_misses_total",
			Help: "Total number of template cache misses",
		},
		[]string{"template"},
	)

	cacheSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "template_cache_size_bytes",
			Help: "Size of cached templates in bytes",
		},
		[]string{"template"},
	)

	cacheItems = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "template_cache_items_total",
			Help: "Total number of items in template cache",
		},
	)
)
