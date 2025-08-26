package cache

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
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
	ctx := context.Background()

	// Создаем тестовый трейсер
	tracer := otel.Tracer("test")
	ctx, parentSpan := tracer.Start(ctx, "TestCacheWithTracing")
	defer parentSpan.End()

	t.Run("Tracing for cache operations", func(t *testing.T) {
		key := "test_tracing"
		value := []byte("test_value")

		// Тест Set
		cache.Set(key, value)

		// Тест Get с трейсингом
		ctx, getSpan := tracer.Start(ctx, "Get")
		got, err := cache.Get(ctx, key)
		getSpan.End()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if string(got) != string(value) {
			t.Errorf("Expected %s, got %s", string(value), string(got))
		}

		// Тест Delete с трейсингом
		ctx, deleteSpan := tracer.Start(ctx, "Delete")
		cache.Delete(ctx, key)
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
	cache := NewCacheWithMetrics(1*time.Second, hits, misses, size, itemsCount)
	ctx := context.Background()

	t.Run("Multiple large items", func(t *testing.T) {
		cache.Clear(ctx) // Clear cache before test

		dataSize := 512 * 1024 // 512KB
		itemCount := 5

		for i := 0; i < itemCount; i++ {
			data := make([]byte, dataSize)
			_, err := rand.Read(data)
			if err != nil {
				t.Fatalf("Failed to generate random data: %v", err)
			}

			key := fmt.Sprintf("large_data_%d", i)
			cache.Set(key, data)
		}

		countValue := testutil.ToFloat64(itemsCount)
		if countValue != float64(itemCount) {
			t.Errorf("Expected %d items, got %v", itemCount, countValue)
		}

		var totalSize float64
		for i := 0; i < itemCount; i++ {
			key := fmt.Sprintf("large_data_%d", i)
			sizeValue := testutil.ToFloat64(size.WithLabelValues(key))
			totalSize += sizeValue
		}

		expectedSize := float64(dataSize * itemCount)
		if totalSize != expectedSize {
			t.Errorf("Expected total size %v, got %v", expectedSize, totalSize)
		}
	})
}

func TestCacheUnderLoad(t *testing.T) {
	_, hits, misses, size, itemsCount := createTestMetrics()
	cache := NewCacheWithMetrics(5*time.Second, hits, misses, size, itemsCount)
	ctx := context.Background()

	// Создаем тестовый трейсер
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(ctx, "TestCacheUnderLoad")
	defer span.End()

	t.Run("Concurrent operations", func(t *testing.T) {
		var wg sync.WaitGroup
		operationCount := 1000
		goroutineCount := 10
		errChan := make(chan error, operationCount*goroutineCount)

		// Запускаем несколько горутин для параллельных операций
		for g := 0; g < goroutineCount; g++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				// Каждая горутина выполняет множество операций
				for i := 0; i < operationCount; i++ {
					key := fmt.Sprintf("key_%d_%d", goroutineID, i)
					value := []byte(fmt.Sprintf("value_%d_%d", goroutineID, i))

					// Записываем значение
					cache.Set(key, value)

					// Читаем значение
					retrieved, err := cache.Get(ctx, key)
					if err != nil {
						errChan <- fmt.Errorf("failed to get value for key %s: %v", key, err)
						continue
					}

					// Проверяем корректность значения
					if !bytes.Equal(value, retrieved) {
						errChan <- fmt.Errorf("value mismatch for key %s", key)
					}

					// Иногда удаляем значение
					if i%3 == 0 {
						cache.Delete(ctx, key)

						// Проверяем что значение удалено
						_, err := cache.Get(ctx, key)
						if err == nil {
							errChan <- fmt.Errorf("key %s should be deleted", key)
						}
					}
				}
			}(g)
		}

		// Ждем завершения всех горутин
		wg.Wait()
		close(errChan)

		// Проверяем наличие ошибок
		var errors []error
		for err := range errChan {
			errors = append(errors, err)
		}

		if len(errors) > 0 {
			for _, err := range errors {
				t.Errorf("Operation error: %v", err)
			}
		}
	})

	t.Run("Load with expiration", func(t *testing.T) {
		cache := NewCache(50 * time.Millisecond)
		var wg sync.WaitGroup
		operationCount := 100
		goroutineCount := 5

		// Запускаем горутины для работы с истекающими значениями
		for g := 0; g < goroutineCount; g++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for i := 0; i < operationCount; i++ {
					key := fmt.Sprintf("exp_key_%d_%d", goroutineID, i)
					value := []byte(fmt.Sprintf("exp_value_%d_%d", goroutineID, i))

					// Записываем значение
					cache.Set(key, value)

					// Случайная задержка
					time.Sleep(time.Duration(30+rand.Intn(40)) * time.Millisecond)

					// Пытаемся получить значение
					_, err := cache.Get(ctx, key)
					if err != nil {
						// Ожидаем ошибок для истекших значений
						continue
					}
				}
			}(g)
		}

		wg.Wait()
	})

	t.Run("Mixed operations under load", func(t *testing.T) {
		var wg sync.WaitGroup
		operationCount := 500
		readerCount := 5
		writerCount := 3
		cleanerCount := 2

		// Запускаем писателей
		for w := 0; w < writerCount; w++ {
			wg.Add(1)
			go func(writerID int) {
				defer wg.Done()

				for i := 0; i < operationCount; i++ {
					key := fmt.Sprintf("mixed_key_%d_%d", writerID, i)
					value := []byte(fmt.Sprintf("mixed_value_%d_%d", writerID, i))
					cache.Set(key, value)
					time.Sleep(time.Millisecond)
				}
			}(w)
		}

		// Запускаем читателей
		for r := 0; r < readerCount; r++ {
			wg.Add(1)
			go func(readerID int) {
				defer wg.Done()

				for i := 0; i < operationCount; i++ {
					for w := 0; w < writerCount; w++ {
						key := fmt.Sprintf("mixed_key_%d_%d", w, i)
						cache.Get(ctx, key)
					}
					time.Sleep(time.Millisecond)
				}
			}(r)
		}

		// Запускаем чистильщиков
		for c := 0; c < cleanerCount; c++ {
			wg.Add(1)
			go func(cleanerID int) {
				defer wg.Done()

				for i := 0; i < operationCount/10; i++ {
					for w := 0; w < writerCount; w++ {
						if rand.Intn(2) == 0 {
							key := fmt.Sprintf("mixed_key_%d_%d", w, i)
							cache.Delete(ctx, key)
						}
					}
					time.Sleep(5 * time.Millisecond)
				}
			}(c)
		}

		wg.Wait()
	})
}
