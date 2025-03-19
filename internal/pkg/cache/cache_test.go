package cache

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.opentelemetry.io/otel"
)

func TestCache(t *testing.T) {
	_, hits, misses, size, itemsCount := createTestMetrics()
	cache := NewCacheWithMetrics(100*time.Millisecond, hits, misses, size, itemsCount)
	ctx := context.Background()

	// Создаем тестовый трейсер
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(ctx, "TestCache")
	defer span.End()

	// Тест добавления и получения значения
	t.Run("Set and Get", func(t *testing.T) {
		key := "test_key"
		value := []byte("test_value")

		cache.Set(key, value)

		got, err := cache.Get(ctx, key)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if string(got) != string(value) {
			t.Errorf("Expected %s, got %s", string(value), string(got))
		}
	})

	// Тест истечения срока жизни значения
	t.Run("Expiration", func(t *testing.T) {
		key := "test_expiration"
		value := []byte("test_value")

		cache.Set(key, value)
		time.Sleep(150 * time.Millisecond)

		_, err := cache.Get(ctx, key)
		if err == nil {
			t.Error("Expected error for expired key")
		}
	})

	// Тест удаления значения
	t.Run("Delete", func(t *testing.T) {
		key := "test_delete"
		value := []byte("test_value")

		cache.Set(key, value)
		cache.Delete(ctx, key)

		_, err := cache.Get(ctx, key)
		if err == nil {
			t.Error("Expected error for deleted key")
		}
	})

	// Тест автоматической очистки
	t.Run("Cleanup", func(t *testing.T) {
		key := "test_cleanup"
		value := []byte("test_value")

		cache.Set(key, value)
		time.Sleep(150 * time.Millisecond)

		_, err := cache.Get(ctx, key)
		if err == nil {
			t.Error("Expected error for cleaned up key")
		}
	})
}

func TestCacheConcurrency(t *testing.T) {
	_, hits, misses, size, itemsCount := createTestMetrics()
	cache := NewCacheWithMetrics(100*time.Millisecond, hits, misses, size, itemsCount)
	ctx := context.Background()
	done := make(chan bool)

	// Создаем тестовый трейсер
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(ctx, "TestCacheConcurrency")
	defer span.End()

	// Тест параллельного доступа
	t.Run("Concurrent access", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			go func(id int) {
				key := "test_key"
				value := []byte("test_value")

				cache.Set(key, value)
				if _, err := cache.Get(ctx, key); err != nil {
					t.Errorf("Expected no error, got %v", err)
				}

				done <- true
			}(i)
		}

		// Ждем завершения всех горутин
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestCacheWithTracing(t *testing.T) {
	_, hits, misses, size, itemsCount := createTestMetrics()
	cache := NewCacheWithMetrics(100*time.Millisecond, hits, misses, size, itemsCount)
	baseCtx := context.Background()

	// Создаем тестовый трейсер
	tracer := otel.Tracer("test")
	ctx, parentSpan := tracer.Start(baseCtx, "TestCacheWithTracing")
	defer parentSpan.End()

	t.Run("Tracing for cache operations", func(t *testing.T) {
		key := "test_tracing"
		value := []byte("test_value")

		// Тест Set
		cache.Set(key, value)

		// Тест Get с трейсингом
		getCtx, getSpan := tracer.Start(ctx, "Get")
		got, err := cache.Get(getCtx, key)
		getSpan.End()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if string(got) != string(value) {
			t.Errorf("Expected %s, got %s", string(value), string(got))
		}

		// Тест Delete с трейсингом
		deleteCtx, deleteSpan := tracer.Start(ctx, "Delete")
		cache.Delete(deleteCtx, key)
		deleteSpan.End()

		// Проверяем, что значение удалено
		_, err = cache.Get(ctx, key)
		if err == nil {
			t.Error("Expected error for deleted key")
		}
	})
}

func TestCacheErrors(t *testing.T) {
	_, hits, misses, size, itemsCount := createTestMetrics()
	cache := NewCacheWithMetrics(100*time.Millisecond, hits, misses, size, itemsCount)
	ctx := context.Background()

	// Создаем тестовый трейсер
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(ctx, "TestCacheErrors")
	defer span.End()

	t.Run("Get non-existent key", func(t *testing.T) {
		_, err := cache.Get(ctx, "non_existent_key")
		if err == nil {
			t.Error("Expected error for non-existent key")
		}
	})

	t.Run("Get after expiration", func(t *testing.T) {
		key := "expiring_key"
		value := []byte("test_value")

		cache.Set(key, value)
		time.Sleep(150 * time.Millisecond)

		_, err := cache.Get(ctx, key)
		if err == nil {
			t.Error("Expected error for expired key")
		}
	})
}

func TestCacheMetrics(t *testing.T) {
	_, hits, misses, size, itemsCount := createTestMetrics()
	cache := NewCacheWithMetrics(100*time.Millisecond, hits, misses, size, itemsCount)
	ctx := context.Background()

	t.Run("Hit metrics", func(t *testing.T) {
		key := "test_hit"
		value := []byte("test_value")

		cache.Set(key, value)
		_, err := cache.Get(ctx, key)
		if err != nil {
			t.Errorf("Failed to get value: %v", err)
		}

		hitValue := testutil.ToFloat64(hits)
		if hitValue != 1 {
			t.Errorf("Expected 1 hit, got %v", hitValue)
		}
	})

	t.Run("Miss metrics", func(t *testing.T) {
		_, err := cache.Get(ctx, "non_existent")
		if err == nil {
			t.Error("Expected error for non-existent key")
		}

		missValue := testutil.ToFloat64(misses)
		if missValue != 1 {
			t.Errorf("Expected 1 miss, got %v", missValue)
		}
	})

	t.Run("Size metrics", func(t *testing.T) {
		key := "test_size"
		value := []byte("test_value")

		cache.Set(key, value)

		sizeValue := testutil.ToFloat64(size.WithLabelValues(key))
		if sizeValue != float64(len(value)) {
			t.Errorf("Expected size %v, got %v", len(value), sizeValue)
		}
	})

	t.Run("Items count metrics", func(t *testing.T) {
		cache.Clear(ctx) // Clear cache before test

		key1 := "test_count_1"
		key2 := "test_count_2"
		value := []byte("test_value")

		cache.Set(key1, value)
		cache.Set(key2, value)

		countValue := testutil.ToFloat64(itemsCount)
		if countValue != 2 {
			t.Errorf("Expected 2 items, got %v", countValue)
		}

		cache.Delete(ctx, key1)

		countValue = testutil.ToFloat64(itemsCount)
		if countValue != 1 {
			t.Errorf("Expected 1 item, got %v", countValue)
		}

		cache.Clear(ctx)

		countValue = testutil.ToFloat64(itemsCount)
		if countValue != 0 {
			t.Errorf("Expected 0 items, got %v", countValue)
		}
	})
}

func TestCacheLargeData(t *testing.T) {
	_, hits, misses, size, itemsCount := createTestMetrics()
	testCache := NewCacheWithMetrics(100*time.Millisecond, hits, misses, size, itemsCount)
	ctx := context.Background()

	t.Run("Large data handling", func(t *testing.T) {
		key := "large_data"
		data := make([]byte, 1024*1024) // 1MB
		_, err := rand.Read(data)
		if err != nil {
			t.Fatalf("Failed to generate random data: %v", err)
		}

		testCache.Set(key, data)
		got, err := testCache.Get(ctx, key)
		if err != nil {
			t.Errorf("Failed to get large data: %v", err)
		}
		if !bytes.Equal(got, data) {
			t.Error("Large data mismatch")
		}
	})
}

func TestCacheUnderLoad(t *testing.T) {
	_, hits, misses, size, itemsCount := createTestMetrics()
	testCache := NewCacheWithMetrics(100*time.Millisecond, hits, misses, size, itemsCount)
	ctx := context.Background()

	t.Run("Concurrent access under load", func(t *testing.T) {
		// Генерируем случайные данные для теста
		data := make([]byte, 1024*1024) // 1MB
		_, err := rand.Read(data)
		if err != nil {
			t.Fatalf("Failed to generate random data: %v", err)
		}

		// Создаем горутины для параллельного доступа
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				key := fmt.Sprintf("test_key_%d", id)
				testCache.Set(key, data)
				_, err := testCache.Get(ctx, key)
				if err != nil {
					t.Errorf("Failed to get value for key %s: %v", key, err)
				}
			}(i)
		}
		wg.Wait()
	})
}
