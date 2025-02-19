package cache

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	cacheHits = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cache_hits_total",
		Help: "Number of cache hits",
	})

	cacheMisses = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cache_misses_total",
		Help: "Number of cache misses",
	})

	cacheSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cache_size_bytes",
		Help: "Size of cached items in bytes",
	}, []string{"key"})

	cacheItemsCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cache_items_count",
		Help: "Number of items in cache",
	})
)

func init() {
	prometheus.MustRegister(cacheHits)
	prometheus.MustRegister(cacheMisses)
	prometheus.MustRegister(cacheSize)
	prometheus.MustRegister(cacheItemsCount)
}

// createTestMetrics creates new metrics with a custom registry for testing
func createTestMetrics() (*prometheus.Registry, prometheus.Gauge, prometheus.Gauge, *prometheus.GaugeVec, prometheus.Gauge) {
	reg := prometheus.NewRegistry()

	hits := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cache_hits_total",
		Help: "Number of cache hits",
	})

	misses := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cache_misses_total",
		Help: "Number of cache misses",
	})

	size := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cache_size_bytes",
		Help: "Size of cached items in bytes",
	}, []string{"key"})

	count := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cache_items_count",
		Help: "Number of items in cache",
	})

	reg.MustRegister(hits)
	reg.MustRegister(misses)
	reg.MustRegister(size)
	reg.MustRegister(count)

	return reg, hits, misses, size, count
}
