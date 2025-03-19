package cache

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"pdf-service-go/internal/pkg/tracing"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/attribute"
)

// Item представляет элемент кэша с временем жизни
type Item struct {
	Value      []byte
	Expiration int64
}

// cacheItem представляет элемент кэша
type cacheItem struct {
	data       []byte
	expiration time.Time
}

func (i *cacheItem) isExpired() bool {
	return time.Now().After(i.expiration)
}

// Cache представляет кэш с поддержкой TTL
type Cache struct {
	sync.RWMutex
	items  map[string]*cacheItem
	ttl    time.Duration
	hits   prometheus.Gauge
	misses prometheus.Gauge
	size   *prometheus.GaugeVec
	count  prometheus.Gauge
}

// Lock блокирует кэш для записи
func (c *Cache) Lock() {
	c.RWMutex.Lock()
}

// Unlock разблокирует кэш для записи
func (c *Cache) Unlock() {
	c.RWMutex.Unlock()
}

// RLock блокирует кэш для чтения
func (c *Cache) RLock() {
	c.RWMutex.RLock()
}

// RUnlock разблокирует кэш для чтения
func (c *Cache) RUnlock() {
	c.RWMutex.RUnlock()
}

// NewCache создает новый экземпляр кэша
func NewCache(ttl time.Duration) *Cache {
	return NewCacheWithMetrics(ttl, cacheHits, cacheMisses, cacheSize, cacheItemsCount)
}

func NewCacheWithMetrics(ttl time.Duration, hits prometheus.Gauge, misses prometheus.Gauge, size *prometheus.GaugeVec, count prometheus.Gauge) *Cache {
	return &Cache{
		items:  make(map[string]*cacheItem),
		ttl:    ttl,
		hits:   hits,
		misses: misses,
		size:   size,
		count:  count,
	}
}

// Set добавляет значение в кэш
func (c *Cache) Set(key string, value []byte) {
	c.Lock()
	defer c.Unlock()

	// If key exists, update size metric
	if oldItem, exists := c.items[key]; exists {
		c.size.WithLabelValues(key).Sub(float64(len(oldItem.data)))
	} else {
		c.count.Inc()
	}

	c.items[key] = &cacheItem{
		data:       value,
		expiration: time.Now().Add(c.ttl),
	}
	c.size.WithLabelValues(key).Add(float64(len(value)))
}

// Get получает значение из кэша
func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	c.RLock()
	item, exists := c.items[key]
	c.RUnlock()

	if !exists {
		c.misses.Inc()
		return nil, fmt.Errorf("key not found: %s", key)
	}

	if item.isExpired() {
		c.Delete(ctx, key)
		c.misses.Inc()
		return nil, fmt.Errorf("key expired: %s", key)
	}

	c.hits.Inc()
	return item.data, nil
}

// Delete удаляет значение из кэша
func (c *Cache) Delete(ctx context.Context, key string) {
	c.Lock()
	defer c.Unlock()

	if item, exists := c.items[key]; exists {
		c.size.WithLabelValues(key).Sub(float64(len(item.data)))
		delete(c.items, key)
		c.count.Dec()
	}
}

// SetFromReader сохраняет данные в кэш из io.Reader
func (c *Cache) SetFromReader(ctx context.Context, key string, reader io.Reader) error {
	ctx, span := tracing.StartSpan(ctx, "Cache.SetFromReader")
	defer span.End()

	span.SetAttributes(attribute.String("cache.key", key))

	// Читаем данные
	data, err := io.ReadAll(reader)
	if err != nil {
		tracing.RecordError(ctx, err)
		return fmt.Errorf("failed to read data: %w", err)
	}

	// Сохраняем в кэш
	c.items[key] = &cacheItem{
		data:       data,
		expiration: time.Now().Add(c.ttl),
	}

	span.AddEvent("Cache updated")
	return nil
}

// Clear очищает весь кэш
func (c *Cache) Clear(ctx context.Context) {
	c.Lock()
	defer c.Unlock()

	for key := range c.items {
		c.size.WithLabelValues(key).Set(0)
	}
	c.items = make(map[string]*cacheItem)
	c.count.Set(0)
}
