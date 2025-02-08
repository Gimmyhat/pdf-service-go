package cache

import (
	"sync"
	"time"
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
func (c *Cache) Get(key string) ([]byte, bool) {
	item, exists := c.items.Load(key)
	if !exists {
		return nil, false
	}

	cacheItem := item.(Item)
	if time.Now().UnixNano() > cacheItem.Expiration {
		c.items.Delete(key)
		return nil, false
	}

	return cacheItem.Value, true
}

// Delete удаляет значение из кэша
func (c *Cache) Delete(key string) {
	c.items.Delete(key)
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
