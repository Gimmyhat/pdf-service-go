package cache

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"pdf-service-go/internal/pkg/tracing"

	"go.opentelemetry.io/otel/attribute"
)

// Item представляет элемент кэша с временем жизни
type Item struct {
	Value      []byte
	Expiration int64
}

// Cache представляет кэш с поддержкой TTL
type Cache struct {
	items sync.Map
	ttl   time.Duration
}

// cacheItem представляет элемент кэша
type cacheItem struct {
	data       []byte
	expiration time.Time
}

// NewCache создает новый экземпляр кэша
func NewCache(ttl time.Duration) *Cache {
	cache := &Cache{
		ttl: ttl,
	}
	go cache.startCleanupTimer()
	return cache
}

// Set добавляет значение в кэш
func (c *Cache) Set(key string, value []byte) {
	c.items.Store(key, Item{
		Value:      value,
		Expiration: time.Now().Add(c.ttl).UnixNano(),
	})
}

// Get получает значение из кэша
func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	ctx, span := tracing.StartSpan(ctx, "Cache.Get")
	defer span.End()

	span.SetAttributes(attribute.String("cache.key", key))

	item, exists := c.items.Load(key)
	if !exists {
		err := fmt.Errorf("cache miss: key %s not found", key)
		tracing.RecordError(ctx, err)
		return nil, err
	}

	cacheItem := item.(Item)
	if time.Now().UnixNano() > cacheItem.Expiration {
		c.items.Delete(key)
		err := fmt.Errorf("cache miss: key %s expired", key)
		tracing.RecordError(ctx, err)
		return nil, err
	}

	span.AddEvent("Cache hit")
	return cacheItem.Value, nil
}

// Delete удаляет значение из кэша
func (c *Cache) Delete(ctx context.Context, key string) {
	ctx, span := tracing.StartSpan(ctx, "Cache.Delete")
	defer span.End()

	span.SetAttributes(attribute.String("cache.key", key))

	c.items.Delete(key)
	span.AddEvent("Cache entry deleted")
}

// startCleanupTimer запускает периодическую очистку устаревших элементов
func (c *Cache) startCleanupTimer() {
	ticker := time.NewTicker(c.ttl)
	for range ticker.C {
		now := time.Now().UnixNano()
		c.items.Range(func(key, value interface{}) bool {
			item := value.(Item)
			if now > item.Expiration {
				c.items.Delete(key)
			}
			return true
		})
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
	c.items.Store(key, cacheItem{
		data:       data,
		expiration: time.Now().Add(c.ttl),
	})

	span.AddEvent("Cache updated")
	return nil
}

// Clear очищает весь кэш
func (c *Cache) Clear(ctx context.Context) {
	ctx, span := tracing.StartSpan(ctx, "Cache.Clear")
	defer span.End()

	c.items = sync.Map{}
	span.AddEvent("Cache cleared")
}
